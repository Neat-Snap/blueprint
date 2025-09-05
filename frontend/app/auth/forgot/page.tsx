"use client";

import React, { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { getMe, requestPasswordReset, beginGoogleLogin } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardDescription, CardTitle } from "@/components/ui/card";
import Link from "next/link";

export default function ForgotPasswordPage() {
  const router = useRouter();
  const params = useSearchParams();
  const [email, setEmail] = useState(() => params.get("email") || "");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);
  const [serverMsg, setServerMsg] = useState<string | null>(null);
  const [oauthOnly, setOauthOnly] = useState(false);
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
    try {
      const res = await requestPasswordReset(email);
      setServerMsg(res.message || null);
      setSent(true);
    } catch (err: unknown) {
      const e = err as { response?: { status?: number; data?: { message?: string } }; message?: string };
      const status = e.response?.status;
      const msg = e.response?.data?.message || e.message || "Failed to request password reset";
      const isOauth = status === 401 || /oauth|google/i.test(msg || "");
      setOauthOnly(isOauth);
      
      setError(isOauth ? null : msg);
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
                  ? "We sent you a password reset link"
                  : oauthOnly
                    ? "This account uses Google sign-in"
                    : "Enter your email address and we’ll send you a reset link"}
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
                    <Link href="/auth/login" className="text-sm underline underline-offset-4">Back to login</Link>
                  </div>
                </div>
              ) : oauthOnly ? (
                <div className="grid gap-4">
                  <div className="rounded-md border border-blue-200 bg-blue-50 p-3 text-blue-800 dark:border-blue-800 dark:bg-blue-950/30 dark:text-blue-200">
                    <div className="flex items-start gap-2">
                      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="mt-0.5">
                        <circle cx="12" cy="12" r="10"></circle>
                        <line x1="12" y1="8" x2="12" y2="12"></line>
                        <line x1="12" y1="16" x2="12.01" y2="16"></line>
                      </svg>
                      <p className="text-sm">This email is registered with Google. Continue with Google to access your account.</p>
                    </div>
                  </div>
                  <Button type="button" variant="outline" className="w-full flex items-center justify-center gap-2" onClick={beginGoogleLogin}>
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                      <path d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"/>
                    </svg>
                    Continue with Google
                  </Button>
                  <div className="text-center text-sm">
                    Not your account? <Link href="/auth/login" className="underline underline-offset-4">Back to login</Link>
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
                  {oauthOnly ? (
                    <Button type="button" variant="outline" className="w-full flex items-center justify-center gap-2" onClick={beginGoogleLogin}>
                      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                        <path d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"/>
                      </svg>
                      Continue with Google
                    </Button>
                  ) : (
                    <Button type="submit" className="w-full" disabled={loading}>
                      {loading ? "Sending..." : "Send reset link"}
                    </Button>
                  )}
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
