"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useTranslations } from "next-intl";

import { beginSignup, getMe } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function SignupPage() {
  const t = useTranslations("Auth.Signup");
  const router = useRouter();
  const [redirecting, setRedirecting] = useState(false);

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

  const handleContinue = () => {
    setRedirecting(true);
    beginSignup();
  };

  return (
    <div className="bg-muted flex min-h-svh flex-col items-center justify-center gap-6 p-6 md:p-10">
      <div className="flex w-full max-w-sm flex-col gap-6">
        <Card>
          <CardHeader className="text-center space-y-1">
            <CardTitle className="text-xl">{t("title")}</CardTitle>
            <CardDescription>{t("subtitle")}</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-6">
            <Button onClick={handleContinue} className="w-full" disabled={redirecting}>
              {redirecting ? t("redirecting") : t("continue")}
            </Button>
            <p className="text-center text-sm text-muted-foreground">
              {t("haveAccount")}{" "}
              <Link href="/auth/login" className="underline underline-offset-4">
                {t("login")}
              </Link>
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
