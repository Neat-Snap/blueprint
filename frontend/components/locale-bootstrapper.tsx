"use client";

import { useEffect, useRef } from "react";
import { getPreferences } from "@/lib/account";
import { usePathname, useRouter } from "next/navigation";

function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp("(?:^|; )" + name.replace(/([.$?*|{}()\[\]\\\/\+^])/g, "\\$1") + "=([^;]*)"));
  return match ? decodeURIComponent(match[1]) : null;
}

export default function LocaleBootstrapper() {
  const router = useRouter();
  const pathname = usePathname();
  const ran = useRef(false);

  useEffect(() => {
    if (ran.current) return;
    ran.current = true;

    (async () => {
      try {
        const current = getCookie("NEXT_LOCALE");
        const prefs = await getPreferences();
        const prefLang = prefs.language;

        if (!prefLang) return; // nothing to sync
        if (current === prefLang) return; // already in sync

        // Set cookie for 180 days
        const maxAge = 60 * 60 * 24 * 180;
        document.cookie = `NEXT_LOCALE=${prefLang}; Path=/; Max-Age=${maxAge}`;

        // Reload current route so server picks up the cookie and renders in new locale
        if (pathname) router.replace(pathname);
        router.refresh();
      } catch {
        // ignore if unauthenticated or request fails
      }
    })();
  }, [pathname, router]);

  return null;
}
