import { cookies, headers } from "next/headers";

export const locales = ["en", "ru", "zh"] as const;
export type Locale = typeof locales[number];
export const defaultLocale: Locale = "en";

export async function resolveLocale(): Promise<Locale> {
  const c = await cookies();
  const cookieLocale = c.get("NEXT_LOCALE")?.value;
  if (cookieLocale && locales.includes(cookieLocale as Locale)) {
    return cookieLocale as Locale;
  }
  // Fallback to Accept-Language
  const h = await headers();
  const accept = h.get("accept-language") || "";
  const first = accept.split(",")[0]?.split("-")[0];
  if (first && locales.includes(first as Locale)) return first as Locale;
  return defaultLocale;
}

export async function loadMessages(locale: Locale) {
  switch (locale) {
    case "ru":
      return (await import("./messages/ru.json")).default;
    case "zh":
      return (await import("./messages/zh.json")).default;
    default:
      return (await import("./messages/en.json")).default;
  }
}
