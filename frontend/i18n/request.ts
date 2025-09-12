import {getRequestConfig} from 'next-intl/server';
import {cookies, headers} from 'next/headers';

const SUPPORTED = new Set(['en', 'ru', 'zh']);

export default getRequestConfig(async () => {
  // Prefer user-set cookie
  const c = await cookies();
  let locale = c.get('NEXT_LOCALE')?.value;

  // Fallback to Accept-Language
  if (!locale) {
    const h = await headers();
    const al = h.get('accept-language') || '';
    const first = al.split(',')[0]?.split('-')[0];
    if (first && SUPPORTED.has(first)) locale = first;
  }

  if (!locale || !SUPPORTED.has(locale)) locale = 'en';

  // Load messages with safe fallback to English
  let messages: Record<string, any>;
  try {
    messages = (await import(`../messages/${locale}.json`)).default;
  } catch {
    messages = (await import(`../messages/en.json`)).default;
  }

  return {locale, messages};
});
