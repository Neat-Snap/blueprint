"use client";

import React, { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { getOverview, type Overview } from "@/lib/dashboard";
import { createWorkspace, getWorkspaceOverview, type WorkspaceOverview } from "@/lib/workspaces";
import { useWorkspace } from "@/lib/workspace-context";

export default function DashboardPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [wsName, setWsName] = useState("");
  const [creating, setCreating] = useState(false);
  const { current } = useWorkspace();
  const [wsOverview, setWsOverview] = useState<WorkspaceOverview | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const data = await getOverview();
        setOverview(data);
      } catch {
        router.replace("/auth/login");
        return;
      } finally {
        setLoading(false);
      }
    })();
  }, [router]);

  useEffect(() => {
    (async () => {
      if (!current) {
        setWsOverview(null);
        return;
      }
      try {
        const data = await getWorkspaceOverview(current.id);
        setWsOverview(data);
      } catch {
        // ignore; auth/layout handles redirects
      }
    })();
  }, [current]);

  const greeting = useMemo(() => {
    if (!overview) return "";
    return overview.user.name || overview.user.email || "";
  }, [overview]);

  const workspaces = useMemo(() => overview?.workspaces ?? [], [overview]);

  async function onCreateWorkspace(e: React.FormEvent) {
    e.preventDefault();
    if (!wsName.trim()) return;
    setCreating(true);
    try {
      const created = await createWorkspace(wsName.trim());
      setWsName("");
      // Optimistic refresh
      setOverview((prev) =>
        prev
          ? {
              ...prev,
              workspaces: [...prev.workspaces, { id: created.id, name: created.name, role: created.role as any }],
              stats: {
                total_workspaces: prev.stats.total_workspaces + 1,
                owner_workspaces: prev.stats.owner_workspaces + (created.role === "owner" ? 1 : 0),
              },
            }
          : prev
      );
    } finally {
      setCreating(false);
    }
  }

  if (loading) return null;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Welcome{greeting ? `, ${greeting}` : ""}</h1>
        <p className="text-muted-foreground">Here’s what’s happening in your workspaces.</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>Total workspaces</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-semibold">{overview?.stats.total_workspaces ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Owned by you</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-semibold">{overview?.stats.owner_workspaces ?? 0}</div>
          </CardContent>
        </Card>
        {wsOverview && (
          <Card>
            <CardHeader>
              <CardTitle>{wsOverview.workspace.name}: Members</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-semibold">{wsOverview.stats.members_count}</div>
              <div className="text-xs text-muted-foreground">Current workspace</div>
            </CardContent>
          </Card>
        )}
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Workspaces</CardTitle>
          </CardHeader>
          <CardContent>
            {workspaces.length ? (
              <ul className="divide-y">
                {workspaces.map((w) => (
                  <li key={w.id} className="flex items-center justify-between py-3">
                    <div>
                      <div className="font-medium">{w.name}</div>
                      <div className="text-xs text-muted-foreground">Role: {w.role}</div>
                    </div>
                    <Button variant="outline" size="sm" onClick={() => router.push(`/dashboard/workspaces/${w.id}`)}>
                      Open
                    </Button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground">No workspaces yet. Create one to get started.</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Create workspace</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={onCreateWorkspace} className="space-y-3">
              <Input
                placeholder="Workspace name"
                value={wsName}
                onChange={(e) => setWsName(e.target.value)}
              />
              <Button type="submit" disabled={creating || !wsName.trim()} className="w-full">
                {creating ? "Creating..." : "Create"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
