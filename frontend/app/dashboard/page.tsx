"use client";

import React, { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { getMe, logout } from "@/lib/auth";

export default function DashboardPage() {
  const router = useRouter();
  const [name, setName] = useState<string | undefined>(undefined);
  const [email, setEmail] = useState<string | undefined>(undefined);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const me = await getMe();
        setName(me.name);
        setEmail(me.email);
      } catch {
        router.replace("/auth/login");
        return;
      } finally {
        setLoading(false);
      }
    })();
  }, [router]);

  if (loading) return null;

  return (
    <main className="p-6 space-y-4">
      <h1 className="text-2xl font-bold">Welcome{(name || email) ? `, ${name || email}` : ""}</h1>
      <Button
        onClick={async () => {
          await logout();
          router.replace("/auth/login");
        }}
      >
        Logout
      </Button>
    </main>
  );
}
