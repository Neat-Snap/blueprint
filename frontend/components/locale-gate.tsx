"use client";

import { useEffect, useRef, useState } from "react";
import { getPreferences } from "@/lib/account";
import { usePathname, useRouter } from "next/navigation";
import { LoadingScreen } from "@/components/loading-screen";

function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp("(?:^|; )" + name.replace(/([.$?*|{}()\[\]\\\/\+^])/g, "\\$1") + "=([^;]*)"));
  return match ? decodeURIComponent(match[1]) : null;
}

export default function LocaleGate({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const ran = useRef(false);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (ran.current) return;
    ran.current = true;

    (async () => {
      try {
        const current = getCookie("NEXT_LOCALE");
        if (current) {
          setReady(true);
          return;
        }

        const prefs = await getPreferences();
        const prefLang = prefs.language;
        if (!prefLang) {
          setReady(true);
          return;
        }

        const maxAge = 60 * 60 * 24 * 180;
        document.cookie = `NEXT_LOCALE=${prefLang}; Path=/; Max-Age=${maxAge}`;

        if (pathname) router.replace(pathname);
      } catch {

    } finally {

        setReady(true);
      }
    })();
  }, [pathname, router]);

  if (!ready) {
    return <LoadingScreen label="Loading workspace" />;
  }

  return <>{children}</>;
}
