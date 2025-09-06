"use client";

import React, { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { confirmEmail, resendEmail } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import Link from "next/link";

export default function VerifyEmailPage() {
  const router = useRouter();
  const params = useSearchParams();

  const [code, setCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [shake, setShake] = useState(false);
  const [resendCountdown, setResendCountdown] = useState<number>(60);
  const [resending, setResending] = useState(false);
  const [confirmationId, setConfirmationId] = useState<string>("");

  const confirmation_id = params.get("cid") || "";
  const email = params.get("email") || "";

  // Keep local state in sync with URL param for confirmation id
  useEffect(() => {
    setConfirmationId(confirmation_id || "");
  }, [confirmation_id]);

  // If we have an email but no confirmation id in the URL, request a new code
  // and update the URL to include the fresh confirmation id.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!email || confirmation_id) return;
      try {
        setResending(true);
        const res = await resendEmail(email);
        if (!cancelled) {
          setConfirmationId(res.confirmation_id);
          const target = `/auth/verify?email=${encodeURIComponent(email)}&cid=${encodeURIComponent(res.confirmation_id)}`;
          router.replace(target);
        }
      } catch (e) {
        // Surface a friendly error; user can try using the explicit resend link
        setError("Could not request a new code. Please try again.");
      } finally {
        setResending(false);
      }
    })();
    return () => { cancelled = true };
  }, [email, confirmation_id, router]);

  useEffect(() => {
    setResendCountdown(60);
    const interval = setInterval(() => {
      setResendCountdown((s) => {
        if (s <= 1) {
          clearInterval(interval);
          return 0;
        }
        return s - 1;
      });
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await confirmEmail(confirmationId || confirmation_id, code);
      router.push("/dashboard");
    } catch (err: unknown) {
      const e = err as { response?: { data?: { message?: string } }; message?: string };
      const msg = e.response?.data?.message || e.message || "Invalid or expired code";
      setError(msg);
      setShake(true);
      setTimeout(() => setShake(false), 300);
    } finally {
      setLoading(false);
    }
  }

  async function onResend() {
    if (!email || resending) return;
    setResending(true);
    setError(null);
    try {
      const res = await resendEmail(email);
      setConfirmationId(res.confirmation_id);
      setResendCountdown(60);
      router.replace(`/auth/verify?email=${encodeURIComponent(email)}&cid=${encodeURIComponent(res.confirmation_id)}`);
    } catch (err: unknown) {
      const e = err as { response?: { data?: { message?: string } }; message?: string };
      const msg = e.response?.data?.message || e.message || "Could not resend email";
      setError(msg);
      setShake(true);
      setTimeout(() => setShake(false), 300);
    } finally {
      setResending(false);
    }
  }

  return (
    <div className="min-h-dvh flex items-center justify-center p-4">
      <Card className={`w-full max-w-sm ${shake ? "animate-shake" : ""}`}>
        <CardHeader>
          <h1 className="text-xl font-semibold">Verify your email</h1>
          <p className="text-sm text-muted-foreground">We sent a code to {email || "your email"}.</p>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="code">Confirmation code</Label>
              <Input id="code" inputMode="numeric" value={code} onChange={(e) => setCode(e.target.value)} required />
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
            <Button type="submit" className="w-full" disabled={loading || !(confirmationId || confirmation_id)}>
              {loading ? "Verifying..." : "Verify"}
            </Button>
            <div className="text-center text-sm text-muted-foreground">
              Didn’t receive the code?
              {" "}
              {resendCountdown > 0 ? (
                <span>
                  You can resend in {Math.floor(resendCountdown / 60)}:{String(resendCountdown % 60).padStart(2, "0")}
                </span>
              ) : (
                <Button type="button" variant="link" className="p-0 h-auto text-primary" onClick={onResend} disabled={resending || !email}>
                  {resending ? "Sending…" : "Resend email"}
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
