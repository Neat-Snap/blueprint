import { NextResponse } from "next/server";

export function middleware() {
  // Disabled: client-side LocaleGate will handle language sync and show the progress bar
  return NextResponse.next();
}

export const config = {
  matcher: [],
};
