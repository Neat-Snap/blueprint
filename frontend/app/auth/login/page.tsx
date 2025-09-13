"use client";

import React, { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { login, beginGoogleLogin, beginGithubLogin, getMe } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardDescription, CardTitle } from "@/components/ui/card";
import Link from "next/link";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Eye, EyeOff } from "lucide-react";
import { GalleryVerticalEnd } from "lucide-react";
import { useTranslations } from "next-intl";

export default function LoginPage() {
  const t = useTranslations('Auth.Login');
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
      const msg = e.response?.data?.message || e.message || t('errors.invalidCredentials');
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
              <CardTitle className="text-xl">{t('title')}</CardTitle>
              <CardDescription>{t('subtitle')}</CardDescription>
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
                    {t('continueWithGoogle')}
                  </Button>
                  <Button
                    variant="outline"
                    className="w-full"
                    onClick={() => {
                      beginGithubLogin();
                    }}
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      viewBox="0 0 24 24"
                      aria-hidden="true"
                      focusable="false"
                      fill="currentColor"
                    >
                      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.3 3.438 9.8 8.207 11.387.6.107.82-.26.82-.58 0-.287-.01-1.05-.016-2.06-3.338.726-4.042-1.61-4.042-1.61-.547-1.39-1.336-1.76-1.336-1.76-1.09-.746.083-.73.083-.73 1.204.084 1.84 1.237 1.84 1.237 1.07 1.835 2.807 1.305 3.492.998.107-.776.418-1.305.762-1.605-2.665-.303-5.466-1.332-5.466-5.93 0-1.31.468-2.38 1.236-3.22-.124-.304-.536-1.527.117-3.183 0 0 1.008-.322 3.3 1.23.957-.266 1.983-.398 3.003-.403 1.02.005 2.046.137 3.005.403 2.29-1.552 3.297-1.23 3.297-1.23.655 1.656.243 2.88.12 3.183.77.84 1.235 1.91 1.235 3.22 0 4.61-2.804 5.624-5.475 5.92.43.37.814 1.103.814 2.226 0 1.606-.015 2.9-.015 3.294 0 .32.218.693.826.576C20.565 21.796 24 17.298 24 12 24 5.37 18.63 0 12 0z"/>
                    </svg>
                    Continue with GitHub
                  </Button>
                </div>
                <div className="after:border-border relative text-center text-sm after:absolute after:inset-0 after:top-1/2 after:z-0 after:flex after:items-center after:border-t">
                  <span className="bg-card text-muted-foreground relative z-10 px-2">
                    {t('orContinueWith')}
                  </span>
                </div>
                <form onSubmit={onSubmit} className="grid gap-6">
                  <div className="grid gap-3">
                    <Label htmlFor="email">{t('email')}</Label>
                    <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required placeholder="m@example.com" />
                  </div>
                  <div className="grid gap-3">
                    <div className="flex items-center">
                      <Label htmlFor="password">{t('password')}</Label>
                      <Link href={`/auth/forgot?email=${encodeURIComponent(email)}`} className="ml-auto text-sm underline-offset-4 hover:underline">
                        {t('forgotPassword')}
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
                        aria-label={showPassword ? t('hidePassword') : t('showPassword')}
                      >
                        {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                      </Button>
                    </div>
                  </div>
                  {error && (
                    <Alert variant="destructive">
                      <AlertTitle>{t('errors.signInFailed')}</AlertTitle>
                      <AlertDescription>{error}</AlertDescription>
                    </Alert>
                  )}
                  <Button type="submit" className="w-full" disabled={loading}>
                    {loading ? t('signingIn') : t('login')}
                  </Button>
                  <div className="text-center text-sm">
                    {t('noAccount')}{" "}
                    <Link href="/auth/signup" className="underline underline-offset-4">
                      {t('signUp')}
                    </Link>
                  </div>
                </form>
              </div>
            </CardContent>
          </Card>
          <div className="text-muted-foreground *:[a]:hover:text-primary text-center text-xs text-balance *:[a]:underline *:[a]:underline-offset-4">
            {t.rich('terms', {
              tos: (chunks) => <a href="#">{chunks}</a>,
              pp: (chunks) => <a href="#">{chunks}</a>,
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
