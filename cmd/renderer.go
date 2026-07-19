package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	runenv "github.com/jlkendrick/grimoire/internal/runenv"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

const previewMaxRunes = 80

// renderer owns everything a ritual run shows on the terminal. All
// observer callbacks and declared prints route through it: graph scopes
// fire events concurrently, and a stdout print landing mid-frame of the
// stderr status line would garble both, so one mutex serializes the lot
// and every write clears the live line first and redraws it after.
//
// Display modes:
//   - one spell in flight on a TTY: the classic sequential UX — header,
//     spinner with the spell's latest stderr line as hint, a "→ preview"
//     line on finish.
//   - overlapping spells on a TTY: one shared status line ("3 running ·
//     lint: …") plus a named static completion line per spell. A spell
//     that ever ran concurrently keeps the named style even if it
//     finishes alone.
//   - stderr not a TTY: static header/finish lines only; chatter is
//     swallowed (piped logs would flood otherwise).
type renderer struct {
	mu    sync.Mutex
	out   io.Writer // stdout: declared prints only
	errW  io.Writer // stderr: headers, previews, footer
	isTTY bool

	total    int // spell-steps in the pipeline: the N of "step k/N"
	started  int // the k
	inflight map[int]*spellView
	sp       *utils.Spinner
	statuses []string // distinct runtime versions + cache statuses, first-seen
	seen     map[string]bool
}

type spellView struct {
	name    string
	last    string // latest stderr line
	grouped bool   // overlapped with another spell at some point
}

func newRenderer(out, errW io.Writer, isTTY bool, total int) *renderer {
	return &renderer{
		out:      out,
		errW:     errW,
		isTTY:    isTTY,
		total:    total,
		inflight: make(map[int]*spellView),
		seen:     make(map[string]bool),
	}
}

func (r *renderer) spellStart(id int, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.started++
	r.inflight[id] = &spellView{name: name}
	if len(r.inflight) > 1 {
		for _, v := range r.inflight {
			v.grouped = true
		}
	}

	r.clearLive()
	fmt.Fprintf(r.errW, "%s step %d/%d · %s\n", accent_style("◈"), r.started, r.total, name)
	r.refreshLive()
}

func (r *renderer) spellStderr(id int, line string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, ok := r.inflight[id]
	if !ok {
		return
	}
	v.last = line
	if r.sp != nil {
		r.sp.UpdateHint(r.liveHint(v))
	}
}

func (r *renderer) spellFinish(id int, name string, res runenv.FinishInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	v := r.inflight[id]
	delete(r.inflight, id)
	for _, s := range []string{res.CacheStatus, res.RuntimeVersion} {
		if s != "" && !r.seen[s] {
			r.seen[s] = true
			r.statuses = append(r.statuses, s)
		}
	}

	r.clearLive()
	switch {
	case res.Err != nil:
		fmt.Fprintf(r.errW, "  %s %s\n\n", dim_style("✗"), dim_style(name))
	case v != nil && v.grouped:
		fmt.Fprintf(r.errW, "  %s %s %s\n\n", accent_style("✓"), name, dim_style("→ "+previewOf(res.Out)))
	default:
		fmt.Fprintf(r.errW, "  %s %s\n\n", accent_style("→"), dim_style(previewOf(res.Out)))
	}
	r.refreshLive()
}

// present is the frontend's Present: one declared output, as canonical
// JSON, on stdout — cleared through the live line so a graph-mode print
// completing mid-run doesn't collide with the animation.
func (r *renderer) present(val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearLive()
	fmt.Fprintln(r.out, string(data))
	r.refreshLive()
	return nil
}

// footer prints the closing timing/runtimes line. It also stops any live
// line, so call it (or close) once the run ends.
func (r *renderer) footer(seconds float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearLive()

	parts := []string{fmt.Sprintf("%.2fs", seconds)}
	parts = append(parts, r.statuses...)
	fmt.Fprintf(r.errW, "\n%s %s\n", accent_style("◈"), dim_style(strings.Join(parts, " · ")))
}

// close stops the live line without a footer — the error path.
func (r *renderer) close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearLive()
}

// clearLive and refreshLive must be called with r.mu held.
func (r *renderer) clearLive() {
	if r.sp != nil {
		r.sp.Stop()
		r.sp = nil
	}
}

func (r *renderer) refreshLive() {
	if !r.isTTY || len(r.inflight) == 0 {
		return
	}
	var hint string
	for _, v := range r.inflight {
		hint = r.liveHint(v)
		break
	}
	r.sp = utils.NewSpinner("")
	r.sp.Start(hint)
}

// liveHint renders the live line's text: the spell's own chatter when
// solo, "N running · spell: chatter" when grouped.
func (r *renderer) liveHint(v *spellView) string {
	if len(r.inflight) <= 1 {
		return v.last
	}
	hint := fmt.Sprintf("%d running", len(r.inflight))
	if v.last != "" {
		hint += fmt.Sprintf(" · %s: %s", v.name, v.last)
	}
	return hint
}

// previewOf renders a finished spell's return value as a one-line
// preview, trimmed to previewMaxRunes.
func previewOf(v any) string {
	if v == nil {
		return ""
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return previewLine(data)
}

// previewLine returns the first non-empty line, trimmed and truncated
// with an ellipsis if it overflowed.
func previewLine(output []byte) string {
	for _, raw := range bytes.Split(output, []byte("\n")) {
		line := strings.TrimSpace(string(raw))
		if line == "" {
			continue
		}
		if utf8.RuneCountInString(line) <= previewMaxRunes {
			return line
		}
		runes := []rune(line)
		return string(runes[:previewMaxRunes]) + "…"
	}
	return ""
}
