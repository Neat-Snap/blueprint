"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMe } from "@/lib/auth";

export default function AuthReadyPage() {
  const router = useRouter();

  useEffect(() => {
    (async () => {
      try {
        await getMe();
        router.replace("/dashboard");
      } catch {
        router.replace("/auth/login");
      }
    })();
  }, [router]);

  return null;
}
