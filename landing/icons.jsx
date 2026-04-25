// Rune-style SVG sigils. Simple geometric shapes only.
const Sigil = {
  Book: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M3 4 Q3 3 4 3 H10 V16 H4 Q3 16 3 15 Z"/>
      <path d="M17 4 Q17 3 16 3 H10 V16 H16 Q17 16 17 15 Z"/>
      <path d="M10 3 V16"/>
    </svg>
  ),
  Scroll: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M4 4 H14 A2 2 0 0 1 16 6 V14 A2 2 0 0 1 14 16 H6"/>
      <path d="M4 4 A2 2 0 0 0 2 6 A2 2 0 0 0 4 8 H12"/>
      <path d="M16 16 A2 2 0 0 0 18 14 A2 2 0 0 0 16 12"/>
    </svg>
  ),
  Wand: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M3 17 L14 6"/>
      <path d="M14 3 L14 9 M11 6 L17 6"/>
      <circle cx="14" cy="6" r="1.2" fill="currentColor"/>
    </svg>
  ),
  Potion: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M8 3 H12 V7 L15 13 A3 3 0 0 1 12 17 H8 A3 3 0 0 1 5 13 L8 7 Z"/>
      <path d="M7 3 H13"/>
    </svg>
  ),
  Skull: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M4 9 A6 6 0 0 1 16 9 V13 H13 V16 H7 V13 H4 Z"/>
      <circle cx="7.5" cy="10" r="1.2" fill="currentColor"/>
      <circle cx="12.5" cy="10" r="1.2" fill="currentColor"/>
    </svg>
  ),
  Key: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <circle cx="6" cy="10" r="3"/>
      <path d="M9 10 H17 M14 10 V13 M17 10 V13"/>
    </svg>
  ),
  Eye: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M2 10 Q10 3 18 10 Q10 17 2 10 Z"/>
      <circle cx="10" cy="10" r="2.2"/>
    </svg>
  ),
  Compass: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <circle cx="10" cy="10" r="7"/>
      <path d="M10 5 L12 10 L10 15 L8 10 Z" fill="currentColor"/>
    </svg>
  ),
  Feather: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M4 16 L16 4 Q16 10 12 14 Q8 18 4 16 Z"/>
      <path d="M4 16 L10 10"/>
    </svg>
  ),
  Crystal: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M10 3 L16 9 L10 17 L4 9 Z"/>
      <path d="M4 9 H16 M10 3 L10 17"/>
    </svg>
  ),
  Flame: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M10 3 Q6 8 7 12 A3 3 0 0 0 13 12 Q14 8 10 3 Z"/>
      <path d="M10 9 Q9 11 10 13 Q11 11 10 9 Z" fill="currentColor"/>
    </svg>
  ),
  Gear: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <circle cx="10" cy="10" r="3"/>
      <path d="M10 2 V5 M10 15 V18 M2 10 H5 M15 10 H18 M4.5 4.5 L6.5 6.5 M13.5 13.5 L15.5 15.5 M4.5 15.5 L6.5 13.5 M13.5 6.5 L15.5 4.5"/>
    </svg>
  ),
  Shield: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M10 3 L16 5 V10 Q16 14 10 17 Q4 14 4 10 V5 Z"/>
    </svg>
  ),
  Star: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.4" {...p}>
      <path d="M10 3 L12 8 L17 8.5 L13 12 L14 17 L10 14 L6 17 L7 12 L3 8.5 L8 8 Z"/>
    </svg>
  ),
  Chevron: (p) => (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.6" {...p}>
      <path d="M7 5 L13 10 L7 15"/>
    </svg>
  ),
};

// Ornate horizontal divider — fleuron style
function Fleuron({ color = "#1c1710", width = 240 }) {
  return (
    <svg width={width} height="24" viewBox="0 0 240 24" fill="none" stroke={color} strokeWidth="1.2" style={{display: "block"}}>
      <path d="M0 12 H80"/>
      <path d="M160 12 H240"/>
      <path d="M88 12 Q100 4 112 12 Q120 18 128 12 Q140 4 152 12" fill="none"/>
      <circle cx="120" cy="12" r="2.2" fill={color}/>
      <circle cx="82" cy="12" r="1.6" fill={color}/>
      <circle cx="158" cy="12" r="1.6" fill={color}/>
    </svg>
  );
}

// Large ornamental corner flourish for the scroll
function CornerFlourish({ color = "#8b2e1f", flip = "" }) {
  return (
    <svg width="64" height="64" viewBox="0 0 64 64" fill="none" stroke={color} strokeWidth="1.2"
         style={{transform: flip}}>
      <path d="M4 4 Q24 4 32 12 Q40 20 36 32 Q32 40 24 36 Q16 32 20 24 Q24 20 28 24"/>
      <path d="M4 4 Q4 24 12 32"/>
      <circle cx="28" cy="24" r="1.6" fill={color}/>
      <circle cx="4" cy="4" r="2" fill={color}/>
    </svg>
  );
}

// Wax seal
function WaxSeal({ size = 96, letter = "G" }) {
  return (
    <div style={{
      position: "relative", width: size, height: size,
      filter: "drop-shadow(2px 3px 3px rgba(0,0,0,0.45))"
    }}>
      <svg viewBox="0 0 100 100" width={size} height={size}>
        <defs>
          <radialGradient id="wax" cx="40%" cy="35%" r="70%">
            <stop offset="0%" stopColor="#d04a33"/>
            <stop offset="60%" stopColor="#8b2e1f"/>
            <stop offset="100%" stopColor="#4a1008"/>
          </radialGradient>
          <filter id="rough">
            <feTurbulence baseFrequency="0.9" numOctaves="2"/>
            <feDisplacementMap in="SourceGraphic" scale="3"/>
          </filter>
        </defs>
        <circle cx="50" cy="50" r="44" fill="url(#wax)" filter="url(#rough)"/>
        <circle cx="50" cy="50" r="36" fill="none" stroke="#4a1008" strokeWidth="1.4"/>
        <circle cx="50" cy="50" r="32" fill="none" stroke="#d04a33" strokeWidth="0.6" opacity="0.6"/>
        <text x="50" y="62" textAnchor="middle"
              fontFamily="UnifrakturCook, serif" fontSize="38" fontWeight="700"
              fill="#f2d9b8" opacity="0.92">{letter}</text>
        {/* star points */}
        {[0,72,144,216,288].map(a => (
          <circle key={a}
                  cx={50 + 28*Math.cos((a-90)*Math.PI/180)}
                  cy={50 + 28*Math.sin((a-90)*Math.PI/180)}
                  r="1.4" fill="#f2d9b8" opacity="0.7"/>
        ))}
      </svg>
    </div>
  );
}

// The grimoire crest — big emblem for hero
function Crest() {
  return (
    <svg viewBox="0 0 220 220" width="220" height="220" style={{display: "block"}}>
      <defs>
        <radialGradient id="crestBg" cx="50%" cy="45%" r="55%">
          <stop offset="0%" stopColor="#3a3020"/>
          <stop offset="100%" stopColor="#0d0a06"/>
        </radialGradient>
        <linearGradient id="goldLine" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#e8c670"/>
          <stop offset="50%" stopColor="#a87a2e"/>
          <stop offset="100%" stopColor="#6b4e1a"/>
        </linearGradient>
      </defs>
      {/* outer circle */}
      <circle cx="110" cy="110" r="104" fill="url(#crestBg)" stroke="url(#goldLine)" strokeWidth="2"/>
      <circle cx="110" cy="110" r="96" fill="none" stroke="#a87a2e" strokeWidth="0.6" opacity="0.6"/>
      {/* ring of runes */}
      {Array.from({length: 16}).map((_, i) => {
        const a = (i / 16) * 2 * Math.PI - Math.PI/2;
        const r = 88;
        const x = 110 + Math.cos(a)*r;
        const y = 110 + Math.sin(a)*r;
        const rune = ["ᚠ","ᚢ","ᚦ","ᚨ","ᚱ","ᚲ","ᚷ","ᚹ","ᚺ","ᚾ","ᛁ","ᛃ","ᛇ","ᛈ","ᛉ","ᛊ"][i];
        return (
          <text key={i} x={x} y={y+5} textAnchor="middle"
                fontFamily="serif" fontSize="13" fill="#d4a84a" opacity="0.85"
                transform={`rotate(${(i/16)*360} ${x} ${y})`}>{rune}</text>
        );
      })}
      {/* inner pentacle + book */}
      <circle cx="110" cy="110" r="70" fill="none" stroke="#a87a2e" strokeWidth="0.8" opacity="0.7"/>
      {/* open book */}
      <g transform="translate(110 110)">
        {/* pages */}
        <path d="M-44 -8 Q-44 -18 -36 -22 L-4 -16 V30 L-36 24 Q-44 22 -44 14 Z"
              fill="#e9dcba" stroke="#1c1710" strokeWidth="1.3"/>
        <path d="M44 -8 Q44 -18 36 -22 L4 -16 V30 L36 24 Q44 22 44 14 Z"
              fill="#e9dcba" stroke="#1c1710" strokeWidth="1.3"/>
        <path d="M-4 -16 V30 M4 -16 V30" stroke="#1c1710" strokeWidth="1"/>
        {/* page lines */}
        {[-8, 0, 8, 16].map(y => (
          <g key={y}>
            <path d={`M-36 ${y} L-10 ${y+1}`} stroke="#6b5a43" strokeWidth="0.8"/>
            <path d={`M10 ${y+1} L36 ${y}`} stroke="#6b5a43" strokeWidth="0.8"/>
          </g>
        ))}
        {/* star above */}
        <path d="M0 -38 L3 -30 L11 -30 L5 -25 L7 -17 L0 -22 L-7 -17 L-5 -25 L-11 -30 L-3 -30 Z"
              fill="#d4a84a" stroke="#a87a2e" strokeWidth="0.6"/>
      </g>
      {/* crossed wands */}
      <g stroke="#d4a84a" strokeWidth="1.6" fill="none" opacity="0.9">
        <path d="M30 40 L190 180"/>
        <path d="M190 40 L30 180"/>
        <circle cx="30" cy="40" r="2.5" fill="#d4a84a"/>
        <circle cx="190" cy="40" r="2.5" fill="#d4a84a"/>
      </g>
    </svg>
  );
}

Object.assign(window, { Sigil, Fleuron, CornerFlourish, WaxSeal, Crest });
