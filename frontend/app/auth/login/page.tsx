"use client";

import React, { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { login, beginGoogleLogin, getMe } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardDescription, CardTitle } from "@/components/ui/card";
import Link from "next/link";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Eye, EyeOff } from "lucide-react";
import { GalleryVerticalEnd } from "lucide-react";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [shake, setShake] = useState(false);
  const [highlightGoogle, setHighlightGoogle] = useState(false);
  const googleBtnRef = useRef<HTMLButtonElement | null>(null);
  const [showPassword, setShowPassword] = useState(false);

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
    return () => { cancelled = true };
  }, [router]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await login(email, password);
      router.push("/dashboard");
    } catch (err: unknown) {
      const e = err as { response?: { status?: number; data?: { message?: string } }; message?: string };
      const status = e.response?.status;
      const msg = e.response?.data?.message || e.message || "Invalid email or password";
      setError(msg);
      if (status === 409 || /google/i.test(msg)) {
        setHighlightGoogle(true);
        requestAnimationFrame(() => googleBtnRef.current?.focus());
      }
      setShake(true);
      setTimeout(() => setShake(false), 300);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="bg-muted flex min-h-svh flex-col items-center justify-center gap-6 p-6 md:p-10">
      <div className="flex w-full max-w-sm flex-col gap-6">
        <Link href="#" className="flex items-center gap-2 self-center font-medium">
          <div className="bg-primary text-primary-foreground flex size-6 items-center justify-center rounded-md">
            <GalleryVerticalEnd className="size-4" />
          </div>
          StatGrad
        </Link>
        <div className={`flex flex-col gap-6 ${shake ? "animate-shake" : ""}`}>
          <Card>
            <CardHeader className="text-center">
              <CardTitle className="text-xl">Welcome back</CardTitle>
              <CardDescription>Login with your Google account or email</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-6">
                <div className="flex flex-col gap-4">
                  <Button
                    ref={googleBtnRef}
                    variant="outline"
                    className={`w-full ${highlightGoogle ? "ring-2 ring-blue-500 animate-pulse" : ""}`}
                    onClick={() => {
                      setHighlightGoogle(false);
                      beginGoogleLogin();
                    }}
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
                      <path
                        d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"
                        fill="currentColor"
                      />
                    </svg>
                    Continue with Google
                  </Button>
                </div>
                <div className="after:border-border relative text-center text-sm after:absolute after:inset-0 after:top-1/2 after:z-0 after:flex after:items-center after:border-t">
                  <span className="bg-card text-muted-foreground relative z-10 px-2">
                    Or continue with
                  </span>
                </div>
                <form onSubmit={onSubmit} className="grid gap-6">
                  <div className="grid gap-3">
                    <Label htmlFor="email">Email</Label>
                    <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required placeholder="m@example.com" />
                  </div>
                  <div className="grid gap-3">
                    <div className="flex items-center">
                      <Label htmlFor="password">Password</Label>
                      <Link href={`/auth/forgot?email=${encodeURIComponent(email)}`} className="ml-auto text-sm underline-offset-4 hover:underline">
                        Forgot your password?
                      </Link>
                    </div>
                    <div className="relative">
                      <Input
                        id="password"
                        type={showPassword ? "text" : "password"}
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        required
                        className="pr-10"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        onClick={() => setShowPassword((s) => !s)}
                        className="absolute right-1 top-1/2 -translate-y-1/2 h-8 w-8 text-muted-foreground"
                        aria-label={showPassword ? "Hide password" : "Show password"}
                      >
                        {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                      </Button>
                    </div>
                  </div>
                  {error && (
                    <Alert variant="destructive">
                      <AlertTitle>Sign in failed</AlertTitle>
                      <AlertDescription>{error}</AlertDescription>
                    </Alert>
                  )}
                  <Button type="submit" className="w-full" disabled={loading}>
                    {loading ? "Signing in..." : "Login"}
                  </Button>
                  <div className="text-center text-sm">
                    Don&apos;t have an account?{" "}
                    <Link href="/auth/signup" className="underline underline-offset-4">
                      Sign up
                    </Link>
                  </div>
                </form>
              </div>
            </CardContent>
          </Card>
          <div className="text-muted-foreground *:[a]:hover:text-primary text-center text-xs text-balance *:[a]:underline *:[a]:underline-offset-4">
            By clicking continue, you agree to our <a href="#">Terms of Service</a>{" "}
            and <a href="#">Privacy Policy</a>.
          </div>
        </div>
      </div>
    </div>
  );
}
