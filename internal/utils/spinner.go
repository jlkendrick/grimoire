package utils

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const spinnerInterval = 100 * time.Millisecond

// Spinner renders a single-line animated status to stderr while a slow
// operation runs. On a TTY it animates in place via carriage return; on a
// non-TTY (piped/redirected stderr) it falls back to printing each hint
// once on its own line so logs stay grep-friendly.
type Spinner struct {
	label    string
	hint     atomic.Value // string
	isTTY    bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	startMu  sync.Mutex
	started  bool
}

func NewSpinner(label string) *Spinner {
	s := &Spinner{
		label: label,
		isTTY: stderrIsTTY(),
	}
	s.hint.Store("")
	return s
}

func (s *Spinner) Start(hint string) {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	if s.started {
		return
	}
	s.started = true
	s.hint.Store(hint)

	if !s.isTTY {
		fmt.Fprintf(os.Stderr, "%s %s %s\n", AccentStyle("◈"), s.label, DimStyle(hint))
		return
	}

	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	go s.animate()
}

func (s *Spinner) UpdateHint(hint string) {
	s.hint.Store(hint)
	if !s.isTTY {
		fmt.Fprintf(os.Stderr, "%s %s %s\n", AccentStyle("◈"), s.label, DimStyle(hint))
	}
}

func (s *Spinner) Stop() {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	if !s.started {
		return
	}
	s.started = false

	if !s.isTTY {
		return
	}

	close(s.stopCh)
	<-s.doneCh
	fmt.Fprint(os.Stderr, "\r\033[K")
}

func (s *Spinner) animate() {
	defer close(s.doneCh)
	ticker := time.NewTicker(spinnerInterval)
	defer ticker.Stop()
	i := 0
	for {
		s.render(spinnerFrames[i%len(spinnerFrames)])
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			i++
		}
	}
}

func (s *Spinner) render(frame string) {
	hint, _ := s.hint.Load().(string)
	fmt.Fprintf(os.Stderr, "\r\033[K%s %s %s %s",
		AccentStyle("◈"),
		s.label,
		AccentStyle("["+frame+"]"),
		DimStyle(hint),
	)
}

func stderrIsTTY() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
