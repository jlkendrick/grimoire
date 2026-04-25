// Low-poly-ish CSS 3D rotating grimoire book + pixel wizard.
// Early-internet "spinning trophy" energy — unapologetically looped.

/* ---------------- ROTATING BOOK ---------------- */
// Six faces built in CSS. Cover has pentacle sigil. Spine embossed "G".
function RotatingBook({ size = 200 }) {
  const W = size * 0.76;   // width  (x)
  const H = size;          // height (y)
  const D = size * 0.22;   // depth  (z)  -- spine thickness

  const face = (tx, ty, tz, rx, ry, extra) => ({
    position: "absolute",
    left: "50%", top: "50%",
    transform: `translate(-50%, -50%) translate3d(${tx}px, ${ty}px, ${tz}px) rotateX(${rx}deg) rotateY(${ry}deg)`,
    backfaceVisibility: "hidden",
    ...extra,
  });

  const coverBg =
    "radial-gradient(ellipse at 30% 20%, #6a3a22 0%, #4a1f10 45%, #2a0f08 100%)";
  const pageEdgeBg =
    "repeating-linear-gradient(90deg, #e9dcba 0, #e9dcba 1px, #c6ac78 1px, #c6ac78 2px)";
  const pageEdgeBgV =
    "repeating-linear-gradient(0deg, #e9dcba 0, #e9dcba 1px, #c6ac78 1px, #c6ac78 2px)";

  return (
    <div style={{
      width: size, height: size,
      perspective: 900,
      perspectiveOrigin: "50% 40%",
    }}>
      <div style={{
        position: "relative", width: "100%", height: "100%",
        transformStyle: "preserve-3d",
        animation: "book-spin 12s linear infinite",
      }}>
        {/* FRONT COVER */}
        <div style={face(0, 0, D/2, 0, 0, {
          width: W, height: H,
          background: coverBg,
          border: "2px solid #1a0a04",
          boxShadow: "inset 0 0 20px rgba(0,0,0,0.6), inset 0 0 0 4px rgba(168,122,46,0.12)",
        })}>
          {/* gold trim */}
          <div style={{
            position: "absolute", inset: 8,
            border: "1px solid #a87a2e",
            boxShadow: "inset 0 0 0 1px rgba(212,168,74,0.3)",
          }}/>
          {/* pentacle */}
          <svg viewBox="0 0 100 100" style={{
            position: "absolute", inset: "50%", width: "60%", height: "60%",
            transform: "translate(-50%, -50%)",
          }}>
            <circle cx="50" cy="50" r="34" fill="none" stroke="#d4a84a" strokeWidth="1.2"
                    style={{filter: "drop-shadow(0 0 3px rgba(212,168,74,0.6))"}}/>
            <circle cx="50" cy="50" r="28" fill="none" stroke="#a87a2e" strokeWidth="0.6"/>
            <path d="M50 20 L62 58 L30 34 L70 34 L38 58 Z"
                  fill="none" stroke="#d4a84a" strokeWidth="1.1"
                  style={{filter: "drop-shadow(0 0 2px rgba(212,168,74,0.5))"}}/>
            {/* runes */}
            {[0, 90, 180, 270].map(a => (
              <text key={a} x="50" y="12" textAnchor="middle"
                    fontFamily="serif" fontSize="6" fill="#d4a84a"
                    transform={`rotate(${a} 50 50)`}>
                ᚠᚢᚦᚨ
              </text>
            ))}
          </svg>
          {/* clasp */}
          <div style={{
            position: "absolute", right: -3, top: "42%",
            width: 12, height: 20,
            background: "linear-gradient(90deg, #d4a84a, #8a6020)",
            border: "1px solid #1a0a04",
          }}/>
        </div>

        {/* BACK COVER */}
        <div style={face(0, 0, -D/2, 0, 180, {
          width: W, height: H,
          background: coverBg,
          border: "2px solid #1a0a04",
          boxShadow: "inset 0 0 20px rgba(0,0,0,0.7)",
        })}>
          <div style={{
            position: "absolute", inset: 8,
            border: "1px solid #a87a2e",
          }}/>
        </div>

        {/* SPINE */}
        <div style={face(-W/2, 0, 0, 0, -90, {
          width: D, height: H,
          background: "linear-gradient(90deg, #2a0f08 0%, #4a1f10 50%, #2a0f08 100%)",
          border: "2px solid #1a0a04",
          display: "flex", flexDirection: "column",
          alignItems: "center", justifyContent: "center",
          gap: 6,
        })}>
          {/* horizontal gold bands */}
          {[0,1,2,3,4].map(i => (
            <div key={i} style={{
              width: "80%", height: 1,
              background: "#a87a2e",
              opacity: 0.7,
            }}/>
          ))}
          <div style={{
            fontFamily: "UnifrakturCook, serif",
            color: "#d4a84a",
            fontSize: D * 0.7,
            lineHeight: 1,
            textShadow: "0 0 4px rgba(212,168,74,0.5)",
            marginTop: 2, marginBottom: 2,
          }}>G</div>
          {[0,1,2,3,4].map(i => (
            <div key={i} style={{
              width: "80%", height: 1,
              background: "#a87a2e",
              opacity: 0.7,
            }}/>
          ))}
        </div>

        {/* RIGHT EDGE (page edges — vertical stripes) */}
        <div style={face(W/2, 0, 0, 0, 90, {
          width: D, height: H,
          background: pageEdgeBg,
          borderTop: "2px solid #1a0a04",
          borderBottom: "2px solid #1a0a04",
          boxShadow: "inset 0 0 6px rgba(0,0,0,0.4)",
        })}/>

        {/* TOP EDGE */}
        <div style={face(0, -H/2, 0, 90, 0, {
          width: W, height: D,
          background: pageEdgeBgV,
          borderLeft: "2px solid #1a0a04",
          borderRight: "2px solid #1a0a04",
          boxShadow: "inset 0 0 6px rgba(0,0,0,0.4)",
        })}/>

        {/* BOTTOM EDGE */}
        <div style={face(0, H/2, 0, -90, 0, {
          width: W, height: D,
          background: pageEdgeBgV,
          borderLeft: "2px solid #1a0a04",
          borderRight: "2px solid #1a0a04",
          boxShadow: "inset 0 0 6px rgba(0,0,0,0.4)",
        })}/>
      </div>
      {/* glow plate behind */}
      <div style={{
        position: "absolute", inset: 0,
        background: "radial-gradient(ellipse at 50% 60%, rgba(212,168,74,0.18), transparent 55%)",
        pointerEvents: "none",
        zIndex: -1,
      }}/>
    </div>
  );
}

/* ---------------- LOW-POLY 3D WIZARD (turntable) ---------------- */
// Y2K / OSRS style untextured low-poly figure. Assembled from flat-shaded
// 3D primitives using CSS transforms. Rotates continuously on Y axis.
function PixelWizard({ scale = 5, size }) {
  // keep legacy `scale` API working (old 24*scale × 28*scale footprint)
  const W = size || 24 * scale;
  const H = size || (28 / 24) * W;

  // palette — flat, OSRS-ish
  const C = {
    hat:        "#2a1a42",
    hatLight:   "#3d2960",
    hatDark:    "#160a28",
    robe:       "#3a2558",
    robeLight:  "#4e3570",
    robeDark:   "#1f1230",
    skin:       "#d9b48a",
    skinDark:   "#a8855f",
    beard:      "#e9dcba",
    beardDark:  "#b9a77f",
    staff:      "#6b4e1a",
    staffDark:  "#3f2d0f",
    orb:        "#ffd97a",
    orbMid:     "#f0aa33",
    orbDark:    "#c87a1a",
    shadow:     "rgba(0,0,0,0.35)",
  };

  // Shared helper: a flat quad face. Positioned via transform. Triangular
  // faces use clip-path polygons on a square div.
  const Face = ({ w, h, tf, bg, clip, z = 1, style }) => (
    <div style={{
      position: "absolute",
      left: "50%", top: "50%",
      width: w, height: h,
      marginLeft: -w / 2, marginTop: -h / 2,
      transform: tf,
      transformStyle: "preserve-3d",
      backfaceVisibility: "hidden",
      background: bg,
      clipPath: clip,
      zIndex: z,
      ...style,
    }}/>
  );

  // scale base metric so wizard fills H
  const U = H / 180;  // 1 unit ≈ H/180, chosen so full figure fits

  return (
    <div style={{
      position: "relative",
      width: W, height: H,
      perspective: 800,
      perspectiveOrigin: "50% 55%",
    }}>
      {/* pedestal shadow */}
      <div style={{
        position: "absolute",
        left: "50%", bottom: 4,
        width: 60 * U, height: 14 * U,
        marginLeft: -30 * U,
        background: "radial-gradient(ellipse, rgba(0,0,0,0.55) 0%, transparent 70%)",
        filter: "blur(1px)",
      }}/>

      {/* turntable */}
      <div style={{
        position: "absolute", inset: 0,
        transformStyle: "preserve-3d",
        animation: "wiz-turn 9s linear infinite",
        transformOrigin: "50% 60%",
      }}>

        {/* ---- STAFF (behind, off to side) ---- */}
        {/* staff shaft — 4-sided prism approximated with 4 rectangular faces */}
        {[0, 90, 180, 270].map((ang, i) => (
          <div key={"staff"+i} style={{
            position: "absolute",
            left: "50%", top: "50%",
            width: 3.5 * U, height: 120 * U,
            marginLeft: -1.75 * U, marginTop: -70 * U,
            transform: `translateX(${34 * U}px) translateZ(6px) rotateY(${ang}deg) translateZ(${1.75 * U}px)`,
            background: i % 2 === 0 ? C.staff : C.staffDark,
          }}/>
        ))}
        {/* orb — an octahedron (8 triangular faces via two diamond-clipped squares crossed) */}
        <div style={{
          position: "absolute",
          left: "50%", top: "50%",
          width: 18 * U, height: 18 * U,
          marginLeft: -9 * U, marginTop: -78 * U,
          transform: `translateX(${34 * U}px) translateZ(6px)`,
          transformStyle: "preserve-3d",
        }}>
          {/* 4 triangular faces making front half of octahedron */}
          <div style={{
            position: "absolute", inset: 0,
            background: C.orb,
            clipPath: "polygon(50% 0, 100% 50%, 50% 100%, 0 50%)",
            transform: "rotateY(0deg) translateZ(4px)",
          }}/>
          <div style={{
            position: "absolute", inset: 0,
            background: C.orbMid,
            clipPath: "polygon(50% 0, 100% 50%, 50% 100%, 0 50%)",
            transform: "rotateY(90deg) translateZ(4px)",
          }}/>
          <div style={{
            position: "absolute", inset: 0,
            background: C.orbDark,
            clipPath: "polygon(50% 0, 100% 50%, 50% 100%, 0 50%)",
            transform: "rotateY(180deg) translateZ(4px)",
          }}/>
          <div style={{
            position: "absolute", inset: 0,
            background: C.orbMid,
            clipPath: "polygon(50% 0, 100% 50%, 50% 100%, 0 50%)",
            transform: "rotateY(-90deg) translateZ(4px)",
          }}/>
        </div>

        {/* ---- ROBE (octagonal cone trunk, 6-sided prism for simplicity) ---- */}
        {Array.from({ length: 6 }).map((_, i) => {
          const ang = (i * 360) / 6;
          const bright = (i === 0 || i === 5) ? C.robeLight
                        : (i === 2 || i === 3) ? C.robeDark
                        : C.robe;
          // each face is a trapezoid: narrower at top, wider at bottom
          return (
            <div key={"robe"+i} style={{
              position: "absolute",
              left: "50%", top: "50%",
              width: 34 * U, height: 85 * U,
              marginLeft: -17 * U, marginTop: -2 * U,
              transform: `rotateY(${ang}deg) translateZ(${17 * U}px)`,
              background: bright,
              clipPath: "polygon(28% 0, 72% 0, 100% 100%, 0 100%)",
            }}/>
          );
        })}
        {/* robe bottom disc (hexagon) */}
        <div style={{
          position: "absolute",
          left: "50%", top: "50%",
          width: 56 * U, height: 56 * U,
          marginLeft: -28 * U, marginTop: 55 * U,
          background: C.robeDark,
          clipPath: "polygon(25% 0, 75% 0, 100% 50%, 75% 100%, 25% 100%, 0 50%)",
          transform: "rotateX(90deg)",
        }}/>

        {/* ---- HEAD (boxy, slightly flattened polyhedron — hexagonal prism) ---- */}
        {Array.from({ length: 6 }).map((_, i) => {
          const ang = (i * 360) / 6;
          const bright = (i === 0) ? C.skin
                       : (i === 5 || i === 1) ? C.skin
                       : (i === 2 || i === 4) ? C.skinDark
                       : C.skinDark;
          return (
            <div key={"head"+i} style={{
              position: "absolute",
              left: "50%", top: "50%",
              width: 13 * U, height: 20 * U,
              marginLeft: -6.5 * U, marginTop: -30 * U,
              transform: `rotateY(${ang}deg) translateZ(${11 * U}px)`,
              background: bright,
            }}/>
          );
        })}
        {/* face — only shows on forward face (angle 0) — we add it at rotateY(0) */}
        <div style={{
          position: "absolute",
          left: "50%", top: "50%",
          width: 13 * U, height: 20 * U,
          marginLeft: -6.5 * U, marginTop: -30 * U,
          transform: `rotateY(0deg) translateZ(${11 * U + 0.5}px)`,
          pointerEvents: "none",
        }}>
          {/* eyes (tiny dark squares) */}
          <div style={{position: "absolute", left: "22%", top: "28%", width: "12%", height: "10%", background: "#1c1710"}}/>
          <div style={{position: "absolute", right: "22%", top: "28%", width: "12%", height: "10%", background: "#1c1710"}}/>
        </div>

        {/* ---- BEARD (hexagonal prism below face, triangular base) ---- */}
        {Array.from({ length: 6 }).map((_, i) => {
          const ang = (i * 360) / 6;
          const bright = (i === 0 || i === 5 || i === 1) ? C.beard : C.beardDark;
          return (
            <div key={"beard"+i} style={{
              position: "absolute",
              left: "50%", top: "50%",
              width: 16 * U, height: 18 * U,
              marginLeft: -8 * U, marginTop: -12 * U,
              transform: `rotateY(${ang}deg) translateZ(${10 * U}px)`,
              background: bright,
              clipPath: "polygon(15% 0, 85% 0, 100% 70%, 50% 100%, 0 70%)",
            }}/>
          );
        })}

        {/* ---- HAT (cone — 6 triangular faces around axis) ---- */}
        {Array.from({ length: 6 }).map((_, i) => {
          const ang = (i * 360) / 6;
          const bright = (i === 0 || i === 5) ? C.hatLight
                       : (i === 2 || i === 3) ? C.hatDark
                       : C.hat;
          return (
            <div key={"hat"+i} style={{
              position: "absolute",
              left: "50%", top: "50%",
              width: 22 * U, height: 50 * U,
              marginLeft: -11 * U, marginTop: -78 * U,
              transform: `rotateY(${ang}deg) translateZ(${8 * U}px)`,
              background: bright,
              clipPath: "polygon(50% 0, 100% 100%, 0 100%)",
            }}/>
          );
        })}
        {/* hat brim — flat hexagon ring */}
        <div style={{
          position: "absolute",
          left: "50%", top: "50%",
          width: 32 * U, height: 32 * U,
          marginLeft: -16 * U, marginTop: -32 * U,
          background: C.hatDark,
          clipPath: "polygon(25% 0, 75% 0, 100% 50%, 75% 100%, 25% 100%, 0 50%)",
          transform: "rotateX(90deg)",
        }}/>
        <div style={{
          position: "absolute",
          left: "50%", top: "50%",
          width: 30 * U, height: 30 * U,
          marginLeft: -15 * U, marginTop: -31 * U,
          background: C.hat,
          clipPath: "polygon(25% 0, 75% 0, 100% 50%, 75% 100%, 25% 100%, 0 50%)",
          transform: "rotateX(90deg) translateZ(-0.5px)",
        }}/>
      </div>
    </div>
  );
}

Object.assign(window, { RotatingBook, PixelWizard });
