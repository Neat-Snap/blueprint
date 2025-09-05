"use client";

import React, { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { confirmPasswordReset, getMe } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardDescription, CardTitle } from "@/components/ui/card";

export default function ResetPasswordPage() {
  const router = useRouter();
  const params = useSearchParams();

  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [shake, setShake] = useState(false);

  const resetId = params.get("cid") || "";
  const code = params.get("code") || "";

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
    if (password.length < 8) {
      setError("Password must be at least 8 characters long");
      setLoading(false);
      return;
    }
    if (password !== confirm) {
      setError("Passwords do not match");
      setShake(true);
      setTimeout(() => setShake(false), 300);
      setLoading(false);
      return;
    }
    try {
      await confirmPasswordReset(resetId, code, password);
      // Backend will set cookie and may redirect to /auth/ready; we can simply push there.
      router.push("/auth/ready");
    } catch (err: unknown) {
      const e = err as { response?: { data?: { message?: string } }; message?: string };
      const msg = e.response?.data?.message || e.message || "Failed to reset password";
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
        <CardHeader className="text-center">
          <CardTitle className="text-xl">Set a new password</CardTitle>
          <CardDescription>Enter and confirm your new password</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="password">New password</Label>
              <Input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirm">Confirm password</Label>
              <Input id="confirm" type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} required />
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
            <Button type="submit" className="w-full" disabled={loading || !resetId || !code}>
              {loading ? "Saving..." : "Reset password"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
