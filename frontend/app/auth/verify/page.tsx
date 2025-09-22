"use client";

import React, { useCallback, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";

import { resendEmail, getMe } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";

export default function VerifyEmailPage() {
  const router = useRouter();
  const params = useSearchParams();

  const [status, setStatus] = useState<"idle" | "sending" | "sent" | "error">("idle");
  const [message, setMessage] = useState<string>("Check your inbox for the verification link we emailed you.");
  const [error, setError] = useState<string | null>(null);
  const [shake, setShake] = useState(false);
  const [autoSent, setAutoSent] = useState(false);

  const email = params.get("email") || "";

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const me = await getMe();
        if (!cancelled && (me?.id || me?.email)) {
          router.replace("/dashboard");
        }
      } catch {
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [router]);

  const triggerResend = useCallback(async () => {
    if (!email) {
      setError("Provide an email address to resend the verification link.");
      setStatus("error");
      setShake(true);
      setTimeout(() => setShake(false), 300);
      return;
    }
    setStatus("sending");
    setError(null);
    try {
      const res = await resendEmail(email);
      setMessage(res.message || "Verification email sent. Please check your inbox.");
      setStatus("sent");
    } catch (err: unknown) {
      const e = err as { response?: { data?: { message?: string } }; message?: string };
      const msg = e.response?.data?.message || e.message || "Could not resend verification email";
      setError(msg);
      setStatus("error");
      setShake(true);
      setTimeout(() => setShake(false), 300);
    }
  }, [email]);

  useEffect(() => {
    if (email && !autoSent) {
      setAutoSent(true);
      void triggerResend();
    }
  }, [autoSent, email, triggerResend]);

  return (
    <div className="min-h-dvh flex items-center justify-center p-4">
      <Card className={`w-full max-w-sm ${shake ? "animate-shake" : ""}`}>
        <CardHeader>
          <h1 className="text-xl font-semibold">Verify your email</h1>
          <p className="text-sm text-muted-foreground">
            {email
              ? `We’re waiting for ${email} to be verified. Check your inbox for the WorkOS email and follow the link.`
              : "Check your email for the verification link and return here once you’re done."}
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          {error ? (
            <div role="alert" className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="mt-0.5">
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="8" x2="12" y2="12"></line>
                <line x1="12" y1="16" x2="12.01" y2="16"></line>
              </svg>
              <p>{error}</p>
            </div>
          ) : (
            <div className="rounded-md border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950/30 dark:text-green-300">
              {message}
            </div>
          )}
          <div className="space-y-2 text-sm text-muted-foreground">
            <p>If you already clicked the link, reload this page or return to the app.</p>
            <p>
              Need a fresh email?{" "}
              <Button
                type="button"
                variant="link"
                className="px-0"
                onClick={triggerResend}
                disabled={status === "sending"}
              >
                {status === "sending" ? "Sending…" : "Resend verification"}
              </Button>
            </p>
          </div>
          <Button asChild variant="outline" className="w-full">
            <Link href="/auth/login">Return to login</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
