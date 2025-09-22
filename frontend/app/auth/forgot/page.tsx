"use client";

import React, { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";

import { getMe, requestPasswordReset } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { validateEmail } from "@/lib/validation";

export default function ForgotPasswordPage() {
  const router = useRouter();
  const params = useSearchParams();
  const [email, setEmail] = useState(() => params.get("email") || "");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);
  const [serverMsg, setServerMsg] = useState<string | null>(null);
  const [shake, setShake] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const me = await getMe();
        if (!cancelled && (me?.id || me?.email)) {
          router.replace("/dashboard");
        }
      } catch {}
    })();
    return () => {
      cancelled = true;
    };
  }, [router]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setServerMsg(null);
    const emailErr = validateEmail(email);
    if (emailErr) {
      setError(emailErr);
      setLoading(false);
      return;
    }
    try {
      const res = await requestPasswordReset(email);
      setServerMsg(res.message || null);
      setSent(true);
    } catch (err: unknown) {
      const e = err as { response?: { status?: number; data?: { message?: string } }; message?: string };
      const msg = e.response?.data?.message || e.message || "Failed to request password reset";
      setError(msg);
      setShake(true);
      setTimeout(() => setShake(false), 300);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="bg-muted flex min-h-svh flex-col items-center justify-center gap-6 p-6 md:p-10">
      <div className="flex w-full max-w-sm flex-col gap-6">
        <div className={`flex flex-col gap-6 ${shake ? "animate-shake" : ""}`}>
          <Card>
            <CardHeader className="text-center">
              <CardTitle className="text-xl">Reset your password</CardTitle>
              <CardDescription>
                {sent
                  ? "Check your inbox for the password reset link."
                  : "Enter your email address and we’ll send you a reset link."}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {sent ? (
                <div className="grid place-items-center gap-4 py-2">
                  <div className="flex items-center justify-center">
                    <div className="bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300 inline-flex h-16 w-16 items-center justify-center rounded-full animate-pulse">
                      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="28" height="28" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M20 6L9 17l-5-5" />
                      </svg>
                    </div>
                  </div>
                  <div className="text-center">
                    <p className="text-sm text-muted-foreground">{serverMsg || "Email sent. Please check your inbox."}</p>
                  </div>
                  <div className="flex gap-2">
                    <Link href="/auth/login" className="text-sm underline underline-offset-4">
                      Back to login
                    </Link>
                  </div>
                </div>
              ) : (
                <form onSubmit={onSubmit} className="grid gap-6">
                  <div className="grid gap-3">
                    <Label htmlFor="email">Email</Label>
                    <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required placeholder="m@example.com" />
                  </div>
                  {error && (
                    <div role="alert" className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300">
                      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="mt-0.5">
                        <circle cx="12" cy="12" r="10"></circle>
                        <line x1="12" y1="8" x2="12" y2="12"></line>
                        <line x1="12" y1="16" x2="12.01" y2="16"></line>
                      </svg>
                      <p>{error}</p>
                    </div>
                  )}
                  <Button type="submit" className="w-full" disabled={loading}>
                    {loading ? "Sending..." : "Send reset link"}
                  </Button>
                  <div className="text-center text-sm">
                    Remembered your password? {" "}
                    <Link href="/auth/login" className="underline underline-offset-4">
                      Log in
                    </Link>
                  </div>
                </form>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
