"use client";

import React, { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useWorkspace } from "@/lib/workspace-context";
import { getWorkspaceOverview, type WorkspaceOverview } from "@/lib/workspaces";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { listInvitations } from "@/lib/workspaces";

export default function DashboardPage() {
  const { current } = useWorkspace();
  const [loading, setLoading] = useState(true);
  const [wsOverview, setWsOverview] = useState<WorkspaceOverview | null>(null);
  const [pendingInvites, setPendingInvites] = useState<number>(0);

  useEffect(() => {
    (async () => {
      if (!current) {
        setWsOverview(null);
        setPendingInvites(0);
        setLoading(false);
        return;
      }
      try {
        const data = await getWorkspaceOverview(current.id);
        setWsOverview(data);
        const invites = await listInvitations(current.id);
        const now = Date.now();
        const count = invites.filter((i: any) => i.status === "pending" && new Date(i.expires_at).getTime() > now).length;
        setPendingInvites(count);
      } finally {
        setLoading(false);
      }
    })();
  }, [current]);

  if (loading) return null;

  if (!current) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">No workspace selected</h1>
        <p className="text-sm text-muted-foreground">Select or create a workspace to see its overview.</p>
        <div>
          <Button asChild>
            <Link href="/dashboard/workspaces">Create workspace</Link>
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{wsOverview?.workspace.name || "Workspace"} overview</h1>
        <p className="text-muted-foreground">Key metrics for this workspace.</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>Members</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-semibold">{wsOverview?.stats.members_count ?? 0}</div>
          </CardContent>
        </Card>
        {pendingInvites > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Pending invitations</CardTitle>
            </CardHeader>
            <CardContent className="flex items-end justify-between gap-3">
              <div className="text-3xl font-semibold">{pendingInvites}</div>
              <Button asChild variant="outline">
                <Link href={`/dashboard/settings`}>Manage</Link>
              </Button>
            </CardContent>
          </Card>
        )}
        {/* Add more workspace-specific cards here as needed */}
      </div>
    </div>
  );
}
