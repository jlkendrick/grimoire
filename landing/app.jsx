// Main landing page app for Grimoire.

const { useState: useStateApp } = React;

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "accentColor": "ember",
  "parchmentTone": "warm",
  "displayFont": "UnifrakturCook",
  "showCandleFlicker": true,
  "showRuneRing": true
}/*EDITMODE-END*/;

const ACCENTS = {
  ember:   { accent: "#8b2e1f", accentBright: "#b8442c" },
  moss:    { accent: "#4a5c2e", accentBright: "#6b8a3a" },
  rune:    { accent: "#2d4a5c", accentBright: "#4a6e8a" },
  plum:    { accent: "#5c2e4a", accentBright: "#8a4a6e" },
};
const PARCHMENTS = {
  warm:   { parchment: "#e9dcba", parchmentDeep: "#c6ac78" },
  pale:   { parchment: "#f1e8cb", parchmentDeep: "#d6c196" },
  smoked: { parchment: "#d8c498", parchmentDeep: "#a88a52" },
};

function ScrollCard({ children, style }) {
  // A scroll: rolled wooden dowel at top + bottom, parchment body between,
  // with a shadow at the top/bottom of the parchment suggesting the roll.
  const dowel = (flip) => (
    <div style={{position: "relative", height: 26}}>
      {/* dowel body */}
      <div style={{
        position: "absolute", left: -12, right: -12,
        top: 0, height: 26,
        background:
          "linear-gradient(180deg, #8a6a3a 0%, #6b4e1a 18%, #3f2d0f 50%, #6b4e1a 82%, #8a6a3a 100%)",
        borderRadius: 13,
        boxShadow:
          "inset 0 0 0 1px #2a1a08, " +
          "inset 2px 0 4px rgba(0,0,0,0.6), inset -2px 0 4px rgba(0,0,0,0.6), " +
          (flip ? "0 -4px 8px rgba(0,0,0,0.5)" : "0 4px 8px rgba(0,0,0,0.5)"),
      }}/>
      {/* wood grain */}
      <div style={{
        position: "absolute", left: 0, right: 0, top: 6, height: 14,
        background:
          "repeating-linear-gradient(90deg, transparent 0 24px, rgba(0,0,0,0.18) 24px 25px, transparent 25px 60px, rgba(0,0,0,0.1) 60px 61px)",
        opacity: 0.8, pointerEvents: "none",
      }}/>
      {/* end caps */}
      <div style={{
        position: "absolute", left: -14, top: -2, width: 20, height: 30,
        background: "radial-gradient(ellipse at 60% 40%, #a88450 0%, #6b4e1a 45%, #2a1a08 100%)",
        borderRadius: "50%",
        boxShadow: "inset 0 0 0 1px #2a1a08, 1px 2px 3px rgba(0,0,0,0.5)",
      }}/>
      <div style={{
        position: "absolute", right: -14, top: -2, width: 20, height: 30,
        background: "radial-gradient(ellipse at 40% 40%, #a88450 0%, #6b4e1a 45%, #2a1a08 100%)",
        borderRadius: "50%",
        boxShadow: "inset 0 0 0 1px #2a1a08, -1px 2px 3px rgba(0,0,0,0.5)",
      }}/>
    </div>
  );

  return (
    <div style={{position: "relative", ...style}}>
      {dowel(false)}
      {/* parchment body */}
      <div style={{position: "relative"}}>
        {/* top shadow cast by the dowel onto the parchment */}
        <div style={{
          position: "absolute", left: 0, right: 0, top: 0, height: 18,
          background: "linear-gradient(180deg, rgba(0,0,0,0.45) 0%, rgba(0,0,0,0.15) 45%, transparent 100%)",
          pointerEvents: "none", zIndex: 2,
        }}/>
        <div className="tex-parchment" style={{
          position: "relative",
          padding: "52px 56px 56px",
          boxShadow: "inset 0 0 60px rgba(139,46,31,0.08), 0 10px 30px rgba(0,0,0,0.6)",
        }}>
          {children}
        </div>
        {/* bottom shadow */}
        <div style={{
          position: "absolute", left: 0, right: 0, bottom: 0, height: 18,
          background: "linear-gradient(0deg, rgba(0,0,0,0.45) 0%, rgba(0,0,0,0.15) 45%, transparent 100%)",
          pointerEvents: "none", zIndex: 2,
        }}/>
      </div>
      {dowel(true)}
    </div>
  );
}

/* ---------------- Hero ---------------- */
function Hero() {
  return (
    <header style={{
      position: "relative",
      padding: "36px 48px 48px",
      borderBottom: "2px solid #000",
    }} className="tex-stone">
      {/* decorative vines left/right */}
      <div style={{
        position: "absolute", top: 0, bottom: 0, left: 0, width: 40,
        background:
          "radial-gradient(circle at 20px 30px, #3a3020 3px, transparent 4px)," +
          "radial-gradient(circle at 20px 90px, #3a3020 3px, transparent 4px)," +
          "radial-gradient(circle at 20px 150px, #3a3020 3px, transparent 4px)",
        backgroundSize: "40px 120px",
        opacity: 0.7,
      }}/>
      <div style={{
        position: "absolute", top: 0, bottom: 0, right: 0, width: 40,
        background:
          "radial-gradient(circle at 20px 30px, #3a3020 3px, transparent 4px)," +
          "radial-gradient(circle at 20px 90px, #3a3020 3px, transparent 4px)," +
          "radial-gradient(circle at 20px 150px, #3a3020 3px, transparent 4px)",
        backgroundSize: "40px 120px",
        opacity: 0.7,
      }}/>

      {/* top nav bar */}
      <nav style={{
        display: "flex", justifyContent: "space-between", alignItems: "center",
        padding: "8px 16px", marginBottom: 28,
        background: "rgba(0,0,0,0.5)",
        border: "1px solid #000",
        outline: "1px solid #3b3024", outlineOffset: "-4px",
        fontFamily: "var(--serif-sc)", letterSpacing: "0.08em",
        fontSize: 14, color: "#cfc196",
      }}>
        <div style={{display: "flex", gap: 28}}>
          <a href="#" style={{color: "#cfc196", borderBottom: "none"}}>᛫ Grimoire</a>
          <a href="#quickstart" style={{color: "#cfc196", borderBottom: "none"}}>Quick Start</a>
          <a href="#demo" style={{color: "#cfc196", borderBottom: "none"}}>The Scrying Mirror</a>
          <a href="#commands" style={{color: "#cfc196", borderBottom: "none"}}>Commands</a>
          <a href="#anatomy" style={{color: "#cfc196", borderBottom: "none"}}>Anatomy of a Spell</a>
        </div>
        <div style={{display: "flex", gap: 20, fontSize: 13}}>
          <a href="https://github.com/jlkendrick/grimoire" style={{color: "#cfc196", borderBottom: "none"}}>GitHub ⟶</a>
          <a href="#" style={{color: "#d4a84a", borderBottom: "none"}}>v0.1.0</a>
        </div>
      </nav>

      {/* crest + title */}
      <div style={{
        display: "flex", alignItems: "center", gap: 48,
        maxWidth: 1200, margin: "0 auto",
      }}>
        <div style={{flex: "none", display: "flex", alignItems: "center", justifyContent: "center", width: 220, height: 220}}>
          <RotatingBook size={200}/>
        </div>
        <div style={{flex: 1}}>
          <div style={{
            fontFamily: "var(--serif-sc)", letterSpacing: "0.24em",
            fontSize: 11, color: "#a87a2e", marginBottom: 18,
          }}>
            A DECLARATIVE EXECUTION FRAMEWORK
          </div>
          <h1 style={{
            fontFamily: "var(--display)",
            fontSize: 78, lineHeight: 0.92, margin: 0,
            color: "#e9dcba",
            textShadow: "0 1px 0 #000, 0 0 16px rgba(168,122,46,0.25)",
            letterSpacing: "0.01em",
          }}>
            Grimoire
          </h1>
          <div style={{
            marginTop: 10,
            fontFamily: "var(--display-2)", fontStyle: "italic",
            fontSize: 19, color: "#cfc196", letterSpacing: "0.01em",
          }}>
            “A spellbook for your codebase.”
          </div>
          <p style={{
            fontFamily: "var(--serif)", fontSize: 15, lineHeight: 1.6,
            color: "#b8a876", marginTop: 22, maxWidth: 560,
          }}>
            Write pure business logic in whatever tongue you choose. Describe the
            incantation in a <span className="mono" style={{color: "#d4a84a"}}>spell.yaml</span>, and
            Grimoire forges it into a fully typed CLI — no boilerplate, no argument
            parsing, no plumbing.
          </p>

          <div style={{display: "flex", gap: 10, marginTop: 26}}>
            <a href="#quickstart" className="btn-stone primary" style={{borderBottom: "none"}}>
              ❧ &nbsp; Begin the Ritual
            </a>
            <a href="https://github.com/jlkendrick/grimoire" className="btn-stone" style={{borderBottom: "none"}}>
              ⛯ &nbsp; View on GitHub
            </a>
            <a href="#commands" className="btn-stone" style={{borderBottom: "none"}}>
              ☾ &nbsp; The Codex
            </a>
          </div>

          <div style={{
            marginTop: 22, display: "flex", gap: 22,
            fontFamily: "var(--mono)", fontSize: 11, color: "#6b5a43",
            letterSpacing: "0.06em",
          }}>
            <span>mit license</span>
            <span>go 1.23+</span>
            <span style={{color: "#a87a2e"}}>· work in progress ·</span>
          </div>
        </div>
      </div>
    </header>
  );
}

/* ---------------- Sidebar spellbook ToC ---------------- */
function Sidebar() {
  const group = (title, items) => (
    <div className="sidebox stud-corners" style={{marginBottom: 18}}>
      <i/><i/><i/><i/>
      <h3>{title}</h3>
      <ul>
        {items.map((it, i) => (
          <li key={i}>
            <a href={it.href}>
              <span className="sigil">{it.icon}</span>
              <span>{it.label}</span>
              {it.ext && <span style={{marginLeft: "auto", color: "#6b5a43", fontSize: 12}}>↗</span>}
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
  return (
    <aside style={{width: 260, flex: "none"}}>
      {group("The Codex", [
        { label: "Quick Start",        icon: <Sigil.Scroll/>,  href: "#quickstart" },
        { label: "Installation",       icon: <Sigil.Potion/>,  href: "#install" },
        { label: "The Scrying Mirror", icon: <Sigil.Crystal/>, href: "#demo" },
        { label: "Anatomy of a Spell", icon: <Sigil.Book/>,    href: "#anatomy" },
        { label: "Command Codex",      icon: <Sigil.Wand/>,    href: "#commands" },
        { label: "Runtimes",           icon: <Sigil.Flame/>,   href: "#runtimes" },
      ])}
      {group("Guild Hall", [
        { label: "GitHub",         icon: <Sigil.Shield/>,  href: "https://github.com/jlkendrick/grimoire", ext: true },
        { label: "Issues",         icon: <Sigil.Skull/>,   href: "#", ext: true },
        { label: "Discussions",    icon: <Sigil.Eye/>,     href: "#", ext: true },
        { label: "Release Notes",  icon: <Sigil.Feather/>, href: "#", ext: true },
        { label: "Roadmap",        icon: <Sigil.Compass/>, href: "#", ext: true },
      ])}
      {group("Arcane Lore", [
        { label: "Why Grimoire?",      icon: <Sigil.Star/>, href: "#" },
        { label: "Design Principles",  icon: <Sigil.Key/>,  href: "#" },
        { label: "FAQ",                icon: <Sigil.Eye/>,  href: "#" },
        { label: "Changelog",          icon: <Sigil.Gear/>, href: "#" },
      ])}

      {/* status pane */}
      <div className="sidebox stud-corners" style={{padding: "14px 16px", textAlign: "center"}}>
        <i/><i/><i/><i/>
        <div style={{fontFamily: "var(--serif-sc)", fontSize: 10, color: "#8a7a5a", letterSpacing: "0.14em"}}>
          THE SPELLBOOK IS OPEN
        </div>
        <div style={{
          fontFamily: "var(--display)", fontSize: 28, color: "#d4a84a",
          margin: "6px 0 2px", fontWeight: 400,
        }}>
          217
        </div>
        <div style={{fontFamily: "var(--serif)", fontSize: 12, color: "#b8a876", fontStyle: "italic"}}>
          spells cast this moon
        </div>
      </div>
    </aside>
  );
}

/* ---------------- Quick Start + Install ---------------- */
function QuickStart() {
  return (
    <section id="quickstart" style={{marginBottom: 64}}>
      <ScrollCard>
        <SectionHead eyebrow="Chapter I" title="The Binding Ritual" seal="I"/>
        <p className="drop-cap" style={{fontSize: 15, lineHeight: 1.75, color: "var(--ink)", margin: "0 0 36px"}}>
          To begin, one must first bind the grimoire to the hand that casts. The rite
          is swift — a single invocation to the package-keepers, and the spellbook
          shall answer to your name ever after. No decorators, no imports, no
          framework code shall touch your own manuscripts.
        </p>

        <div style={{display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))", gap: 28, marginTop: 32}}>
          <div>
            <div style={{
              fontFamily: "var(--serif-sc)", fontSize: 11, letterSpacing: "0.16em",
              color: "var(--ember)", marginBottom: 8,
            }}>
              I · VIA HOMEBREW
            </div>
            <div className="codeblock">
              <span className="prompt">$ </span><span className="cmd">brew install</span> <span className="arg">jlkendrick/tap/grimoire</span>
            </div>
          </div>
          <div>
            <div style={{
              fontFamily: "var(--serif-sc)", fontSize: 11, letterSpacing: "0.16em",
              color: "var(--ember)", marginBottom: 8,
            }}>
              II · VIA GO
            </div>
            <div className="codeblock">
              <span className="prompt">$ </span><span className="cmd">go install</span> <span className="arg">github.com/jlkendrick/grimoire@latest</span>
            </div>
          </div>
        </div>

        <div style={{marginTop: 32}}>
          <div style={{
            fontFamily: "var(--serif-sc)", fontSize: 11, letterSpacing: "0.16em",
            color: "var(--ember)", marginBottom: 8,
          }}>
            III · FROM SOURCE
          </div>
          <div className="codeblock">
<span className="prompt">$ </span><span className="cmd">git clone</span> <span className="arg">https://github.com/jlkendrick/grimoire.git</span>
{"\n"}<span className="prompt">$ </span><span className="cmd">cd</span> <span className="arg">grimoire</span> && <span className="cmd">go build</span> <span className="arg">-o grimoire .</span>
{"\n"}<span className="prompt">$ </span><span className="cmd">mv</span> <span className="arg">./grimoire</span> <span className="arg">/usr/local/bin/</span>
{"\n"}<span className="cmt"># confirm the binding</span>
{"\n"}<span className="prompt">$ </span><span className="cmd">grimoire</span> <span className="arg">--version</span>
{"\n"}<span className="str">grimoire 0.3.2 · go1.23.4 · darwin/arm64</span>
          </div>
        </div>

        {/* four-step ritual */}
        <div style={{marginTop: 56}}>
          <div style={{textAlign: "center", marginBottom: 18}}>
            <Fleuron width={180}/>
          </div>
          <h3 style={{
            fontFamily: "var(--display)", fontSize: 26, textAlign: "center",
            color: "var(--ink)", margin: "0 0 36px", fontWeight: 400,
          }}>
            The Four Gestures of Casting
          </h3>
          <div style={{display: "grid", gridTemplateColumns: "1fr 1fr 1fr 1fr", gap: 22}}>
            {[
              { n: "I",   verb: "Scaffold",  code: "grimoire init",       gloss: "Inscribe a fresh spell.yaml in the current directory." },
              { n: "II",  verb: "Bind",      code: "grimoire add f.py:fn", gloss: "Divine the function’s signature and bind it by name." },
              { n: "III", verb: "Register",  code: "grimoire register",    gloss: "Commit the spellbook to your global grimoire." },
              { n: "IV",  verb: "Cast",      code: "grimoire <spell>",     gloss: "Invoke any bound spell from anywhere on the system." },
            ].map((s, i) => (
              <div key={i} style={{
                position: "relative",
                paddingTop: 14,
                borderTop: "1px solid var(--ink)",
              }}>
                <div style={{
                  fontFamily: "var(--display)", fontSize: 22,
                  color: "var(--ember)", lineHeight: 1, marginBottom: 8,
                }}>{s.n}</div>
                <div style={{
                  fontFamily: "var(--serif-sc)", letterSpacing: "0.14em",
                  fontSize: 11, color: "var(--ink)", marginBottom: 12,
                }}>THE {s.verb.toUpperCase()}</div>
                <div className="mono" style={{
                  fontSize: 11.5, color: "#4a3018", marginBottom: 12,
                }}>{s.code}</div>
                <div style={{
                  fontFamily: "var(--serif)", fontSize: 13,
                  color: "var(--ink-soft)", lineHeight: 1.55,
                }}>{s.gloss}</div>
              </div>
            ))}
          </div>
        </div>
      </ScrollCard>
    </section>
  );
}

/* ---------------- Section Head ---------------- */
function SectionHead({ eyebrow, title, seal }) {
  return (
    <div style={{marginBottom: 36, display: "flex", alignItems: "center", gap: 18}}>
      <WaxSeal letter={seal} size={52}/>
      <div style={{flex: 1, minWidth: 0}}>
        <div style={{
          fontFamily: "var(--serif-sc)", letterSpacing: "0.22em",
          fontSize: 11, color: "var(--ember)",
        }}>
          {eyebrow.toUpperCase()}
        </div>
        <h2 style={{
          fontFamily: "var(--display)",
          fontSize: 42, margin: "4px 0 10px",
          color: "var(--ink)", lineHeight: 1,
          fontWeight: 400,
        }}>
          {title}
        </h2>
        <div style={{height: 1, background: "var(--ink)"}}/>
      </div>
    </div>
  );
}

/* ---------------- Wax Seal ---------------- */
function WaxSeal({ letter, size = 52 }) {
  // small ember-colored wax disk with a raised letter in blackletter
  return (
    <div style={{
      position: "relative",
      width: size, height: size,
      flex: "none",
      filter: "drop-shadow(1px 2px 2px rgba(0,0,0,0.45))",
    }}>
      <svg viewBox="0 0 100 100" width={size} height={size} style={{display: "block"}}>
        <defs>
          <radialGradient id={`wax-${letter}`} cx="38%" cy="32%" r="72%">
            <stop offset="0%" stopColor="#d04a33"/>
            <stop offset="55%" stopColor="var(--ember)"/>
            <stop offset="100%" stopColor="#4a0f08"/>
          </radialGradient>
        </defs>
        {/* irregular wax splatter outline */}
        <path d="M50,4 C62,6 72,2 80,10 C92,14 96,26 94,38 C100,48 96,62 88,70 C92,82 82,94 70,94 C62,100 48,98 40,92 C28,96 14,90 10,78 C2,70 4,56 10,48 C4,38 10,22 22,18 C28,8 40,2 50,4 Z"
              fill={`url(#wax-${letter})`}/>
        {/* inner stamped disc */}
        <circle cx="50" cy="50" r="30" fill="none"
                stroke="rgba(0,0,0,0.35)" strokeWidth="1.2"/>
        <circle cx="50" cy="50" r="28" fill="none"
                stroke="rgba(255,220,180,0.15)" strokeWidth="0.8"/>
        {/* highlight */}
        <ellipse cx="36" cy="32" rx="12" ry="7" fill="rgba(255,220,180,0.25)"/>
      </svg>
      {/* letter overlay */}
      <div style={{
        position: "absolute", inset: 0,
        display: "flex", alignItems: "center", justifyContent: "center",
        fontFamily: "var(--display)",
        fontSize: size * 0.46,
        color: "#f5d8b8",
        textShadow: "0 1px 0 rgba(0,0,0,0.6), 0 -1px 0 rgba(255,220,180,0.2)",
        lineHeight: 1,
        paddingBottom: size * 0.04,
      }}>
        {letter}
      </div>
    </div>
  );
}

/* ---------------- Scrying Mirror (demo) ---------------- */
function DemoSection() {
  return (
    <section id="demo" style={{marginBottom: 64}}>
      <ScrollCard>
        <SectionHead eyebrow="Chapter II" title="The Scrying Mirror" seal="II"/>

        {/* Intro row: wizard + prose side by side */}
        <div style={{display: "flex", alignItems: "flex-start", gap: 32, marginBottom: 36}}>
          <div style={{flex: "none", paddingTop: 4}}>
            <PixelWizard scale={5}/>
            <div style={{
              marginTop: 10, textAlign: "center",
              fontFamily: "var(--serif-sc)", fontSize: 10, letterSpacing: "0.12em",
              color: "var(--ink-faded)",
            }}>
              THE CASTER
            </div>
          </div>
          <p style={{fontSize: 15, lineHeight: 1.75, color: "var(--ink)", margin: 0, flex: 1}}>
            Gaze upon the mirror and observe three rites in sequence — the binding of a
            function, the inscription into <span className="mono" style={{fontSize: 13}}>spell.yaml</span>, and
            the casting of the spell itself. Grimoire handles interpreter resolution,
            argument parsing, type coercion, and execution. Your function stays wholly
            uninstrumented.
          </p>
        </div>

        <Terminal/>

        <div style={{
          marginTop: 28, display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 24,
          fontFamily: "var(--serif-sc)", fontSize: 11, letterSpacing: "0.1em",
          color: "var(--ink-soft)",
        }}>
          {[
            ["AUTO VENV",     "pyproject · requirements"],
            ["TYPE COERCION", "argparse, handled"],
            ["CACHED BINARY", "go · compiled once"],
            ["ZERO BOILERPLATE", "no decorators"],
          ].map(([k,v]) => (
            <div key={k} style={{borderTop: "1px solid var(--ink)", paddingTop: 10}}>
              <div style={{color: "var(--ember)"}}>{k}</div>
              <div style={{fontFamily: "var(--serif)", fontStyle: "italic", color: "var(--ink-soft)", letterSpacing: 0, marginTop: 2, fontSize: 13}}>{v}</div>
            </div>
          ))}
        </div>
      </ScrollCard>
    </section>
  );
}

/* ---------------- Anatomy of a Spell ---------------- */
function Anatomy() {
  return (
    <section id="anatomy" style={{marginBottom: 64}}>
      <ScrollCard>
        <SectionHead eyebrow="Chapter III" title="Anatomy of a Spell" seal="III"/>

        <div style={{display: "grid", gridTemplateColumns: "1.05fr 1fr", gap: 48}}>
          <div>
            <p className="drop-cap" style={{fontSize: 15, lineHeight: 1.75, margin: "0 0 18px"}}>
              A <em>spell</em> is a precise recipe. It names the function, points to the
              file that contains it, declares its arguments, and carries all that is
              needed to summon the right interpreter. Once inscribed, any hand bearing
              Grimoire may lift the spell from the page and cast it.
            </p>
            <p style={{fontSize: 14, lineHeight: 1.75, margin: "0 0 24px", color: "var(--ink-soft)"}}>
              Unlike a common script runner, a spell carries the <em>full incantation</em>:
              the function to call, the arguments it expects, the types, the defaults.
              No README reading. No “it works on my machine.”
            </p>

            <div style={{
              borderLeft: "2px solid var(--ember)",
              padding: "4px 16px", margin: "28px 0",
              fontFamily: "var(--display-2)", fontStyle: "italic",
              fontSize: 16, lineHeight: 1.5, color: "var(--ink)",
            }}>
              “The function knows nothing of the framework. The framework knows
              everything of the function.”
              <div style={{
                marginTop: 8, fontFamily: "var(--serif-sc)",
                fontStyle: "normal", fontSize: 10, color: "var(--ink-faded)",
                letterSpacing: "0.14em",
              }}>
                FIRST LAW OF THE GRIMOIRE
              </div>
            </div>

            <ul style={{
              listStyle: "none", padding: 0, margin: 0,
              fontSize: 13.5, lineHeight: 1.6,
            }}>
              {[
                ["Local spellbook",  "spell.yaml committed beside your code — push it and anyone can cast."],
                ["Global grimoire",  "~/.grimoire aggregates old shell scripts, utilities, and one-offs under one CLI."],
                ["Signature divination", "tree-sitter reads your source and extracts types, defaults, and argnames."],
                ["Hybrid execution",     "Python venvs, Go isolated wrapper modules — provisioned, cached, forgotten."],
              ].map(([k, v]) => (
                <li key={k} style={{
                  display: "grid", gridTemplateColumns: "14px 1fr",
                  gap: 10, padding: "10px 0",
                  borderBottom: "1px dotted #9a8660",
                }}>
                  <span style={{color: "var(--ember)", fontSize: 13, lineHeight: 1.4}}>※</span>
                  <div>
                    <strong className="smallcaps" style={{color: "var(--ink)", fontSize: 12, letterSpacing: "0.08em"}}>{k.toUpperCase()}</strong>
                    <div style={{color: "var(--ink-soft)", marginTop: 2}}>{v}</div>
                  </div>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <div style={{
              fontFamily: "var(--serif-sc)", fontSize: 11, letterSpacing: "0.16em",
              color: "var(--ember)", marginBottom: 8,
            }}>
              A SPELL, TRANSCRIBED
            </div>
            <div className="codeblock" style={{fontSize: 12.5}}>
<span className="cmt"># spell.yaml</span>
{"\n"}<span className="key">functions</span>:
{"\n"}  - <span className="key">name</span>:     <span className="val">greet</span>
{"\n"}    <span className="key">path</span>:     <span className="val">scripts/greet.py</span>
{"\n"}    <span className="key">function</span>: <span className="val">say_hello</span>
{"\n"}    <span className="key">args</span>:
{"\n"}      - <span className="key">name</span>: <span className="val">name</span>
{"\n"}        <span className="key">type</span>: <span className="val">str</span>
{"\n"}      - <span className="key">name</span>: <span className="val">times</span>
{"\n"}        <span className="key">type</span>: <span className="val">int</span>
{"\n"}        <span className="key">default</span>: <span className="val">1</span>
            </div>

            <div style={{
              fontFamily: "var(--serif-sc)", fontSize: 11, letterSpacing: "0.16em",
              color: "var(--ember)", margin: "24px 0 8px",
            }}>
              THE GENERATED CLI
            </div>
            <div className="codeblock" style={{fontSize: 12.5}}>
<span className="prompt">$ </span><span className="cmd">grimoire greet</span> <span className="arg">--name</span> <span className="str">"Alice"</span> <span className="arg">--times</span> <span className="str">3</span>
{"\n"}<span style={{color: "#d6c79c"}}>  Hello, Alice!</span>
{"\n"}<span style={{color: "#d6c79c"}}>  Hello, Alice!</span>
{"\n"}<span style={{color: "#d6c79c"}}>  Hello, Alice!</span>
            </div>

            <p style={{
              marginTop: 20, fontFamily: "var(--serif)",
              fontSize: 13, fontStyle: "italic",
              color: "var(--ink-soft)", lineHeight: 1.6,
            }}>
              Your source file <span className="mono" style={{fontStyle: "normal", color: "var(--ember)", fontSize: 12}}>greet.py</span> remains
              untouched — no imports, no decorators, no framework-aware code.
              The spell is a parallel manuscript.
            </p>
          </div>
        </div>
      </ScrollCard>
    </section>
  );
}

/* ---------------- Commands Grimoire ---------------- */
function Commands() {
  const rows = [
    ["grimoire init",                  "Scaffold a spell.yaml in the current directory.",           "I"],
    ["grimoire add <file>:<function>", "Bind a function and auto-extract its signature.",           "II"],
    ["grimoire sync",                  "Regenerate argument signatures for all registered functions.", "III"],
    ["grimoire register [path]",       "Register a project’s spell.yaml with the global grimoire.", "IV"],
    ["grimoire clean [--global]",      "Purge cached venvs for functions whose source has vanished.", "V"],
    ["grimoire <name> [flags]",        "Cast any bound spell by its declared name.",                "VI"],
  ];
  return (
    <section id="commands" style={{marginBottom: 64}}>
      <ScrollCard>
        <SectionHead eyebrow="Chapter IV" title="The Command Codex" seal="IV"/>
        <p style={{fontSize: 15, lineHeight: 1.75, color: "var(--ink)", margin: "0 0 32px", maxWidth: 720}}>
          Six gestures govern the grimoire. Commit them to memory, or keep them
          pinned near your conjuring-desk.
        </p>
        <table className="grimoire">
          <thead>
            <tr>
              <th style={{width: 36}}>№</th>
              <th style={{width: "34%"}}>INCANTATION</th>
              <th>EFFECT</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(([cmd, desc, n], i) => (
              <tr key={i}>
                <td style={{fontFamily: "var(--display)", fontSize: 16, color: "var(--ember)", width: 36}}>{n}</td>
                <td className="cmd">{cmd}</td>
                <td style={{color: "var(--ink-soft)"}}>{desc}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </ScrollCard>
    </section>
  );
}

/* ---------------- Runtimes ---------------- */
function Runtimes() {
  const cards = [
    {
      letter: "P",
      name: "Python",
      lang: "Pythonica",
      status: "Full Support",
      lore: "Automatic virtual-environment provisioning from requirements.txt or pyproject.toml. Tree-sitter divines signatures from source.",
      features: ["venv isolation", "pyproject.toml", "requirements.txt", "system python fallback"],
    },
    {
      letter: "G",
      name: "Go",
      lang: "Golangia",
      status: "Full Support",
      lore: "An isolated wrapper module is forged per project, compiled on first use, and cached. Subsequent castings are near-instantaneous.",
      features: ["isolated wrapper", "compile once", "cached binary", "go 1.23+"],
    },
    {
      letter: "?",
      name: "More",
      lang: "Yet Unwritten",
      status: "Planned",
      lore: "The adapter interface was forged to be language-agnostic from the first line. Rust, Node, Ruby, Bash — all sigils await.",
      features: ["rust", "node.js", "ruby", "bash"],
      ghost: true,
    },
  ];
  return (
    <section id="runtimes" style={{marginBottom: 64}}>
      <ScrollCard>
        <SectionHead eyebrow="Chapter V" title="Runtime Sigils" seal="V"/>
        <p style={{fontSize: 15, lineHeight: 1.75, color: "var(--ink)", margin: "0 0 36px", maxWidth: 720}}>
          Each tongue the grimoire speaks is bound by its own sigil — a small family
          of incantations that know how to summon its interpreter.
        </p>
        <div style={{display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 28}}>
          {cards.map((c, i) => (
            <div key={i} style={{
              position: "relative",
              paddingTop: 14,
              borderTop: "1px solid var(--ink)",
              opacity: c.ghost ? 0.6 : 1,
            }}>
              <div style={{
                display: "flex", alignItems: "baseline", gap: 10, marginBottom: 8,
              }}>
                <div style={{fontFamily: "var(--display)", fontSize: 22, color: "var(--ink)", lineHeight: 1, fontWeight: 400}}>{c.name}</div>
                <div style={{
                  fontFamily: "var(--display-2)", fontStyle: "italic",
                  color: "var(--ink-faded)", fontSize: 13,
                }}>{c.lang}</div>
              </div>

              <div style={{
                fontFamily: "var(--serif-sc)", fontSize: 10, letterSpacing: "0.16em",
                color: c.ghost ? "var(--ink-faded)" : "var(--ember)", marginBottom: 12,
              }}>
                {c.status.toUpperCase()}
              </div>

              <p style={{
                fontSize: 13, lineHeight: 1.65, color: "var(--ink-soft)",
                margin: "0 0 14px",
              }}>{c.lore}</p>

              <ul style={{listStyle: "none", margin: 0, padding: 0, fontSize: 11.5}}>
                {c.features.map(f => (
                  <li key={f} className="mono" style={{
                    padding: "2px 0", color: c.ghost ? "var(--ink-faded)" : "var(--ink)",
                  }}>
                    <span style={{color: "var(--ember)", marginRight: 8}}>›</span>{f}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </ScrollCard>
    </section>
  );
}

/* ---------------- Footer ---------------- */
function Footer() {
  return (
    <footer className="tex-stone" style={{
      position: "relative",
      padding: "48px 48px 32px",
      borderTop: "2px solid #000",
      color: "#cfc196",
    }}>
      <div style={{
        maxWidth: 1200, margin: "0 auto",
        display: "grid", gridTemplateColumns: "1.2fr 1fr 1fr 1fr", gap: 48,
      }}>
        <div>
          <div style={{
            fontFamily: "var(--display)", fontSize: 42, color: "#e9dcba",
            textShadow: "0 2px 0 #000",
          }}>Grimoire</div>
          <p style={{fontSize: 15, color: "#8a7a5a", lineHeight: 1.55, marginTop: 10}}>
            A declarative, language-agnostic execution framework. Written in Go,
            at peace with any tongue.
          </p>
          <div style={{marginTop: 14, fontFamily: "var(--serif-sc)", fontSize: 12, color: "#6b5a43", letterSpacing: "0.1em"}}>
            ◈ MIT · ca. mmxxvi ◈
          </div>
        </div>
        {[
          ["THE CODEX",   ["Quick Start", "Installation", "Commands", "Anatomy of a Spell"]],
          ["GUILD HALL",  ["GitHub", "Issues", "Discussions", "Contribute"]],
          ["ARCANE LORE", ["Changelog", "Roadmap", "FAQ", "License"]],
        ].map(([t, items]) => (
          <div key={t}>
            <div style={{
              fontFamily: "var(--serif-sc)", fontSize: 13, color: "#d4a84a",
              letterSpacing: "0.15em", marginBottom: 14,
            }}>⟡ {t}</div>
            <ul style={{listStyle: "none", padding: 0, margin: 0, fontSize: 15, lineHeight: 2}}>
              {items.map(x => (
                <li key={x}><a href="#" style={{color: "#cfc196", borderBottom: "none"}}>{x}</a></li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      <div style={{
        marginTop: 40, paddingTop: 20,
        borderTop: "1px solid #3b3024",
        textAlign: "center",
        fontFamily: "var(--serif-sc)", fontSize: 12, color: "#6b5a43", letterSpacing: "0.18em",
      }}>
        ✦ ᛋᛈᛖᛚᛚ · ᚱᛖᛊᛈᛟᚾᛋᛁᛒᛚᛖ · ᚲᚨᛋᛏᛁᚾᚷ ✦ &nbsp;&nbsp; · &nbsp;&nbsp; MAY YOUR STACKTRACES BE FEW
      </div>
    </footer>
  );
}

/* ---------------- Root ---------------- */
function App() {
  const [tweaks, setTweaks] = (window.useTweaks || ((d) => [d, () => {}]))(TWEAK_DEFAULTS);

  const accent = ACCENTS[tweaks.accentColor] || ACCENTS.ember;
  const parch  = PARCHMENTS[tweaks.parchmentTone] || PARCHMENTS.warm;

  const rootStyle = {
    "--ember": accent.accent,
    "--ember-bright": accent.accentBright,
    "--parchment": parch.parchment,
    "--parchment-deep": parch.parchmentDeep,
    "--display": `"${tweaks.displayFont}", "IM Fell English", serif`,
  };

  return (
    <div style={rootStyle}>
      <Hero/>
      <div className="hinge"/>

      <div style={{
        display: "flex", gap: 36,
        maxWidth: 1280, margin: "0 auto",
        padding: "40px 48px 20px",
        background: "#0a0806",
        position: "relative",
      }}>
        <Sidebar/>
        <main style={{flex: 1, minWidth: 0}}>
          <QuickStart/>
          <DemoSection/>
          <Anatomy/>
          <Commands/>
          <Runtimes/>
        </main>
      </div>

      <Footer/>

      {window.TweaksPanel && (
        <TweaksPanel title="Tweaks">
          <TweakSection label="Colour & Tone"/>
          <TweakRadio label="Accent sigil" value={tweaks.accentColor}
            options={["ember", "moss", "rune", "plum"]}
            onChange={(v) => setTweaks("accentColor", v)}/>
          <TweakRadio label="Parchment" value={tweaks.parchmentTone}
            options={["warm", "pale", "smoked"]}
            onChange={(v) => setTweaks("parchmentTone", v)}/>
          <TweakSection label="Typography"/>
          <TweakSelect label="Display font" value={tweaks.displayFont}
            options={["UnifrakturCook", "IM Fell English", "Cormorant Garamond"]}
            onChange={(v) => setTweaks("displayFont", v)}/>
          <TweakSection label="Atmosphere"/>
          <TweakToggle label="Candle flicker" value={tweaks.showCandleFlicker}
            onChange={(v) => setTweaks("showCandleFlicker", v)}/>
          <TweakToggle label="Rune ring on crest" value={tweaks.showRuneRing}
            onChange={(v) => setTweaks("showRuneRing", v)}/>
        </TweaksPanel>
      )}
    </div>
  );
}

(function mount() {
  const el = document.getElementById("root");
  if (!el) { window.addEventListener("DOMContentLoaded", mount, { once: true }); return; }
  if (!window.__grimoireRoot) window.__grimoireRoot = ReactDOM.createRoot(el);
  window.__grimoireRoot.render(<App/>);
})();
