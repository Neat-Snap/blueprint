// Server component fallback for Suspense (no client hooks or browser APIs)

export default function LoadingFallback({ label = "Loading" }: { label?: string }) {
  return (
    <div id="lf"
      aria-live="polite"
      aria-busy="true"
      role="status"
      style={{
        position: "fixed",
        inset: 0,
        zIndex: 50,
        display: "grid",
        placeItems: "center",
        // Set explicit dark bg to avoid initial white before CSS variables apply
        background: "#0b0b12",
        // Provide immediate defaults for custom properties
        // @ts-ignore - allow CSS custom properties on style
        ['--bg' as any]: '#0b0b12',
        // @ts-ignore
        ['--track' as any]: '#1f2937',
        // @ts-ignore
        ['--bar' as any]: '#e5e7eb',
      }}
    >
      <style>
        {`
        @keyframes cascade_loadingbar { 
          0% { transform: translateX(-40%); }
          100% { transform: translateX(140%); }
        }
        /* Derive colors from app theme variables when available, with safe fallbacks */
        #lf { 
          --bg: var(--background, #0b0b12);
          --track: var(--muted, #1f2937);
          --bar: var(--primary, #e5e7eb);
        }
        @media (prefers-color-scheme: light) {
          #lf { 
            --bg: var(--background, #ffffff);
            --track: var(--muted, #eef2f7);
            --bar: var(--primary, #111827);
          }
        }
        @media (prefers-color-scheme: dark) {
          #lf { 
            --bg: var(--background, #0b0b12);
            --track: var(--muted, #1f2937);
            --bar: var(--primary, #e5e7eb);
          }
        }
        `}
      </style>
      <div style={{ width: "min(80vw, 360px)", padding: "0 24px" }}>
        {/* Screen-reader label only to stay minimal visually */}
        {label ? <div style={{ position: "absolute", width: 1, height: 1, padding: 0, margin: -1, overflow: "hidden", clip: "rect(0,0,0,0)", whiteSpace: "nowrap", border: 0 }}>{label}</div> : null}
        <div style={{ height: 6, width: "100%", overflow: "hidden", borderRadius: 6, background: "var(--track)" }}>
          <div
            style={{
              height: "100%",
              width: "40%",
              borderRadius: 6,
              background: "var(--bar)",
              animation: "cascade_loadingbar 2.4s linear infinite",
              willChange: "transform",
            }}
          />
        </div>
      </div>
    </div>
  );
}
