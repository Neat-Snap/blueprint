"use client";

import React, { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { resendEmail, confirmEmail, getMe } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import Link from "next/link";

export default function ResendEmailPage() {
  const router = useRouter();
  const params = useSearchParams();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [shake, setShake] = useState(false);
  const [confirmationId, setConfirmationId] = useState<string>("");
  const [code, setCode] = useState("");

  const email = params.get("email") || "";

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const me = await getMe();
        if (!cancelled && (me?.id || me?.email)) {
          router.replace("/dashboard");
        }
      } catch (_) {
      }
    })();
    return () => { cancelled = true };
  }, [router]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!email) {
        setError("Missing email parameter");
        setLoading(false);
        return;
      }
      try {
        const res = await resendEmail(email);
        if (!cancelled) {
          setConfirmationId(res.confirmation_id);
        }
      } catch (err: any) {
        const msg = err?.response?.data?.message || "Could not resend email";
        setError(msg);
        setShake(true);
        setTimeout(() => setShake(false), 300);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true };
  }, [email]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await confirmEmail(confirmationId, code);
      router.push("/dashboard");
    } catch (err: any) {
      const msg = err?.response?.data?.message || "Invalid or expired code";
      setError(msg);
      setShake(true);
      setTimeout(() => setShake(false), 300);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-dvh flex items-center justify-center p-4">
      <Card className={`w-full max-w-sm ${shake ? "animate-shake" : ""}`}>
        <CardHeader>
          <h1 className="text-xl font-semibold">Resend verification email</h1>
          <p className="text-sm text-muted-foreground">{email ? `We\'re sending a new code to ${email}.` : "Provide your email to resend the code."}</p>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex flex-col items-center justify-center py-6 gap-3">
              <svg className="animate-spin h-6 w-6 text-primary" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"></path>
              </svg>
              <p className="text-sm text-muted-foreground">Sending…</p>
            </div>
          ) : error ? (
            <div className="space-y-4">
              <div role="alert" className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300">
                <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="mt-0.5">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="8" x2="12" y2="12"></line>
                  <line x1="12" y1="16" x2="12.01" y2="16"></line>
                </svg>
                <p>{error}</p>
              </div>
              <div className="text-sm">
                <Link href={`/auth/verify?email=${encodeURIComponent(email)}`} className="text-primary hover:underline">Go back</Link>
              </div>
            </div>
          ) : (
            <form onSubmit={onSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="code">Enter the new code</Label>
                <Input id="code" inputMode="numeric" value={code} onChange={(e) => setCode(e.target.value)} required />
              </div>
              <Button type="submit" className="w-full" disabled={loading || !confirmationId}>
                {loading ? "Verifying..." : "Verify"}
              </Button>
              <p className="text-xs text-muted-foreground text-center">Didn't get it? You can try again later.</p>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
