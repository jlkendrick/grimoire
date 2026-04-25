// The Scrying Mirror — terminal demo with animated typing.
// Wrapped in an ornate stone frame with runic inscriptions.

const { useState, useEffect, useRef } = React;

// Colored token helper
const T = ({ c, children }) => <span style={{color: c}}>{children}</span>;

// Individual "scene" frames — the terminal cycles through these
const SCENES = [
  {
    label: "I · The Binding",
    title: "grimoire init",
    lines: [
      { p: "~/projects/ledger", c: "grimoire init" },
      { o: <><T c="#d4a84a">+</T> <T c="#c0b080">Inscribed scroll.yaml</T></> },
      { o: <><T c="#6b5a43">  · ~/projects/ledger/scroll.yaml</T></> },
    ],
  },
  {
    label: "II · The Inscription",
    title: "grimoire add",
    lines: [
      { p: "~/projects/ledger", c: "grimoire add scripts/reconcile.py:run_daily" },
      { o: <><T c="#d4a84a">+</T> <T c="#c0b080">Divining signature...</T></> },
      { o: <><T c="#d4a84a">├──</T> <T c="#c0b080">function </T><T c="#b6c28a">run_daily</T></> },
      { o: <><T c="#d4a84a">├──</T> <T c="#c0b080">args </T><T c="#b6c28a">date:str accounts:list[str] dry_run:bool=True</T></> },
      { o: <><T c="#d4a84a">└──</T> <T c="#c0b080">runtime </T><T c="#b6c28a">python · pyproject.toml</T></> },
      { o: <><T c="#d4a84a">+</T> <T c="#c0b080">Bound to scroll </T><T c="#b8442c">reconcile</T></> },
    ],
  },
  {
    label: "III · The Casting",
    title: "grimoire reconcile",
    lines: [
      { p: "~/projects/ledger", c: "grimoire reconcile --date 2026-04-23 --accounts ops,payroll" },
      { o: <><T c="#d4a84a">◈</T> <T c="#c0b080">provisioning venv </T><T c="#d4a84a">[····]</T> <T c="#6b5a43">cached</T></> },
      { o: <><T c="#d4a84a">◈</T> <T c="#c0b080">casting spell </T><T c="#b8442c">reconcile</T></> },
      { o: "" },
      { o: <T c="#d6c79c">  reconciled 2,104 entries across 2 accounts</T> },
      { o: <T c="#d6c79c">  delta: $−412.18   status:</T> },
      { o: <T c="#b6c28a">  ✓ balanced</T> },
      { o: "" },
      { o: <T c="#6b5a43">◈ 0.42s · cached · python 3.12</T> },
    ],
  },
];

function Terminal() {
  const [sceneIdx, setSceneIdx] = useState(0);
  const [visibleLines, setVisibleLines] = useState(0);
  const [typedCmd, setTypedCmd] = useState("");
  const [phase, setPhase] = useState("typing"); // typing -> output -> pause -> next
  const [manual, setManual] = useState(false);
  const timerRef = useRef(null);

  const scene = SCENES[sceneIdx];
  const cmdLine = scene.lines[0];
  const outputLines = scene.lines.slice(1);

  useEffect(() => {
    if (manual) return;
    clearTimeout(timerRef.current);

    if (phase === "typing") {
      if (typedCmd.length < cmdLine.c.length) {
        timerRef.current = setTimeout(() => {
          setTypedCmd(cmdLine.c.slice(0, typedCmd.length + 1));
        }, 32 + Math.random() * 30);
      } else {
        timerRef.current = setTimeout(() => setPhase("output"), 400);
      }
    } else if (phase === "output") {
      if (visibleLines < outputLines.length) {
        timerRef.current = setTimeout(() => setVisibleLines(v => v + 1), 180);
      } else {
        timerRef.current = setTimeout(() => setPhase("pause"), 1600);
      }
    } else if (phase === "pause") {
      timerRef.current = setTimeout(() => {
        setSceneIdx(i => (i + 1) % SCENES.length);
        setTypedCmd("");
        setVisibleLines(0);
        setPhase("typing");
      }, 1200);
    }

    return () => clearTimeout(timerRef.current);
  }, [phase, typedCmd, visibleLines, sceneIdx, manual]);

  const jumpTo = (i) => {
    clearTimeout(timerRef.current);
    setManual(true);
    setSceneIdx(i);
    setTypedCmd(SCENES[i].lines[0].c);
    setVisibleLines(SCENES[i].lines.length - 1);
    setPhase("pause");
    setTimeout(() => setManual(false), 100);
  };

  return (
    <div style={{position: "relative"}}>
      {/* Ornate stone frame */}
      <div style={{
        background: "linear-gradient(180deg, #2a241c 0%, #14100a 100%)",
        padding: "24px 28px",
        border: "2px solid #000",
        outline: "1px solid #6b5a3b",
        outlineOffset: "-8px",
        boxShadow: "inset 0 0 0 4px #0a0806, inset 0 0 40px rgba(0,0,0,0.7), 0 4px 0 #000",
        position: "relative",
      }} className="stud-corners">
        <i/><i/><i/><i/>

        {/* Top runic inscription */}
        <div style={{
          display: "flex", justifyContent: "space-between", alignItems: "center",
          marginBottom: 14, padding: "0 8px",
          fontFamily: "var(--mono)", fontSize: 11, letterSpacing: "0.15em",
          color: "#a87a2e", textTransform: "uppercase",
        }}>
          <span>✦ ᚦᛖ · ᛋᚲᚱᛁᚾᚷ · ᛗᛁᚱᚱᛟᚱ ✦</span>
          <span style={{color: "#6b5a43"}}>grimoire · v0.3.2</span>
        </div>

        {/* The "glass" / terminal pane */}
        <div style={{
          position: "relative",
          background: "radial-gradient(ellipse at 30% 20%, #0f1418 0%, #050708 100%)",
          border: "1px solid #000",
          outline: "1px solid #2a3b45",
          outlineOffset: "-3px",
          minHeight: 340,
          padding: "18px 22px",
          fontFamily: "var(--mono)",
          fontSize: 14,
          color: "#d6c79c",
          lineHeight: 1.65,
          overflow: "hidden",
          boxShadow: "inset 0 0 60px rgba(0,0,0,0.8), inset 0 0 20px rgba(45,74,92,0.25)",
        }}>
          {/* Subtle scan shimmer */}
          <div style={{
            position: "absolute", inset: 0,
            background: "linear-gradient(transparent 0%, rgba(255,255,255,0.02) 50%, transparent 100%)",
            backgroundSize: "100% 6px",
            pointerEvents: "none",
            mixBlendMode: "overlay",
          }}/>

          {/* Prompt line */}
          <div>
            <T c="#b8442c">❧</T>{" "}
            <T c="#6b8a7a">{cmdLine.p}</T>{" "}
            <T c="#c09a5a">❯</T>{" "}
            <T c="#e8d9a8">{typedCmd}</T>
            {phase === "typing" && <span className="cursor-blink" style={{
              display: "inline-block", width: 8, height: 16,
              background: "#d4a84a", marginLeft: 2, verticalAlign: "-2px"
            }}/>}
          </div>

          {/* Output lines */}
          <div style={{marginTop: 6}}>
            {outputLines.slice(0, visibleLines).map((l, i) => (
              <div key={i}>{l.o || "\u00A0"}</div>
            ))}
            {phase === "output" && visibleLines < outputLines.length && (
              <span className="cursor-blink" style={{
                display: "inline-block", width: 8, height: 16,
                background: "#d4a84a", verticalAlign: "-2px"
              }}/>
            )}
            {phase === "pause" && (
              <div style={{marginTop: 10}}>
                <T c="#6b5a43">▪</T>{" "}
                <span className="cursor-blink" style={{
                  display: "inline-block", width: 8, height: 16,
                  background: "#d4a84a", verticalAlign: "-2px"
                }}/>
              </div>
            )}
          </div>
        </div>

        {/* Bottom chapter dots */}
        <div style={{
          marginTop: 18,
          display: "flex", justifyContent: "center", gap: 28,
        }}>
          {SCENES.map((s, i) => (
            <button key={i} onClick={() => jumpTo(i)} style={{
              background: "none", border: "none", cursor: "pointer",
              fontFamily: "var(--serif-sc)", letterSpacing: "0.1em",
              fontSize: 12, color: i === sceneIdx ? "#d4a84a" : "#6b5a43",
              padding: "6px 10px",
              borderBottom: i === sceneIdx ? "1px solid #d4a84a" : "1px solid transparent",
              transition: "color 0.2s",
            }}>
              {s.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

window.Terminal = Terminal;
