"use client";

import * as React from "react";
import { Progress } from "@/components/ui/progress";

export function LoadingScreen({ label, immediate = false }: { label?: string; immediate?: boolean }) {
  const [value, setValue] = React.useState(0);
  const [visible, setVisible] = React.useState(immediate);

  React.useEffect(() => {
    const clearers: number[] = [];

    const startProgress = () => {
      let v = 8;
      setValue(v);
      const id = window.setInterval(() => {
        v = v + Math.max(1, Math.floor((90 - v) / 8));
        if (v >= 90) {
          v = 90;
          window.clearInterval(id);
        }
        setValue(v);
      }, 120);
      clearers.push(id);
    };

    if (immediate) {
      setVisible(true);
      startProgress();
    } else {
      // Avoid flash: show after 150ms if still loading
      const showTimer = window.setTimeout(() => setVisible(true), 150);
      clearers.push(showTimer);
      // Smooth staged progress that asymptotically approaches 90%
      const startTimer = window.setTimeout(startProgress, 180);
      clearers.push(startTimer);
    }

    return () => {
      for (const c of clearers) {
        // c may be timeout id or interval id
        window.clearTimeout(c);
        window.clearInterval(c);
      }
    };
  }, [immediate]);

  return (
    <div
      className={`fixed inset-0 z-50 grid place-items-center bg-background transition-opacity duration-200 ${
        visible ? "opacity-100" : "opacity-0"
      }`}
      aria-live="polite"
      aria-busy="true"
      role="status"
    >
      <div className="w-[min(80vw,360px)] px-6">
        {label && <div className="sr-only">{label}</div>}
        <Progress value={value} />
      </div>
    </div>
  );
}
