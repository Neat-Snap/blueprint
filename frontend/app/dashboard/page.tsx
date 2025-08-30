"use client";

import React, { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { logout } from "@/lib/auth";
import { getOverview, type DashboardOverviewResponse } from "@/lib/dashboard";

export default function DashboardPage() {
  const router = useRouter();
  const [data, setData] = useState<DashboardOverviewResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const overview = await getOverview();
        setData(overview);
      } catch {
        router.replace("/auth/login");
        return;
      } finally {
        setLoading(false);
      }
    })();
  }, [router]);

  if (loading) return null;
  if (!data) return null;

  const greetingName = data.user?.name || data.user?.email;

  return (
    <main className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Welcome{greetingName ? `, ${greetingName}` : ""}</h1>
        <Button
          onClick={async () => {
            await logout();
            router.replace("/auth/login");
          }}
        >
          Logout
        </Button>
      </div>

      <section className="space-y-2">
        <h2 className="text-lg font-semibold">Your workspaces</h2>
        {data.workspaces.length === 0 ? (
          <p className="text-sm text-muted-foreground">No workspaces yet.</p>
        ) : (
          <ul className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {data.workspaces.map((ws) => (
              <li key={ws.id} className="border rounded-md p-3 flex items-center justify-between">
                <div>
                  <p className="font-medium">{ws.name}</p>
                  <p className="text-xs text-muted-foreground">Role: {ws.role}</p>
                </div>
                {/* Placeholder for future navigation */}
                <Button variant="secondary" onClick={() => router.push(`/dashboard/workspaces/${ws.id}`)}>
                  Open
                </Button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <div className="text-sm text-muted-foreground">
          Total workspaces: {data.stats.total_workspaces} • Owned: {data.stats.owner_workspaces}
        </div>
      </section>
    </main>
  );
}
