"use client";

import React, { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { confirmEmail, getMe } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader } from "@/components/ui/card";

export default function VerifyEmailPage() {
  const router = useRouter();
  const params = useSearchParams();

  const [code, setCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [shake, setShake] = useState(false);

  const confirmation_id = params.get("cid") || "";
  const email = params.get("email") || "";

  // If already authenticated, redirect to dashboard
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const me = await getMe();
        if (!cancelled && (me?.id || me?.email)) {
          router.replace("/dashboard");
        }
      } catch (_) {
        // not logged in; ignore
      }
    })();
    return () => { cancelled = true };
  }, [router]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await confirmEmail(confirmation_id, code);
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
            <Button type="submit" className="w-full" disabled={loading || !confirmation_id}>
              {loading ? "Verifying..." : "Verify"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
