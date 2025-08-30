"use client";

import React, { useEffect, useMemo, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { listNotifications, markNotificationRead, type Notification } from "@/lib/notifications";
import { acceptInvitation } from "@/lib/workspaces";
import { useRouter } from "next/navigation";
import { useWorkspace } from "@/lib/workspace-context";

function parseInviteData(data: string): { workspace_id?: number; workspace_name?: string; token?: string; role?: string } {
  try {
    return JSON.parse(data || "{}");
  } catch {
    return {} as { workspace_id?: number; workspace_name?: string; token?: string; role?: string };
  }
}

export default function NotificationsPage() {
  const router = useRouter();
  const { setCurrentId } = useWorkspace();
  const [loading, setLoading] = useState(true);
  const [list, setList] = useState<Notification[]>([]);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [tab, setTab] = useState<"unread" | "read">("unread");

  useEffect(() => {
    (async () => {
      try {
        const n = await listNotifications();
        setList(n);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  async function onAcceptInvite(n: Notification) {
    const payload = parseInviteData(n.data);
    if (!payload.token) return;
    await acceptInvitation(payload.token);
    await markNotificationRead(n.id);
    setList((prev) => prev.map((x) => (x.id === n.id ? { ...x, readAt: new Date().toISOString() } : x)));
    if (payload.workspace_id) {
      setCurrentId(Number(payload.workspace_id));
      router.push(`/dashboard/settings`);
    }
  }

  async function onMarkRead(n: Notification) {
    await markNotificationRead(n.id);
    setList((prev) => prev.map((x) => (x.id === n.id ? { ...x, readAt: new Date().toISOString() } : x)));
  }

  const unread = useMemo(() => list.filter((n) => !n.readAt), [list]);
  const read = useMemo(() => list.filter((n) => !!n.readAt), [list]);

  if (loading) return null;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Notifications</h1>
        <p className="text-muted-foreground text-sm">Manage your notifications.</p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Inbox</CardTitle>
            <div className="inline-flex rounded-md border p-1 text-sm">
              <button
                className={`px-3 py-1 rounded ${tab === "unread" ? "bg-secondary" : ""}`}
                onClick={() => setTab("unread")}
                type="button"
              >
                Unread ({unread.length})
              </button>
              <button
                className={`px-3 py-1 rounded ${tab === "read" ? "bg-secondary" : ""}`}
                onClick={() => setTab("read")}
                type="button"
              >
                Read ({read.length})
              </button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {(tab === "unread" ? unread : read).length ? (
            <ul className="divide-y">
              {(tab === "unread" ? unread : read).map((n) => {
                const isInvite = n.type === "workspace_invite";
                const isOpen = expandedId === n.id;
                const d = isInvite ? parseInviteData(n.data) : {};
                return (
                  <li key={n.id} className="py-3">
                    <button
                      className="flex w-full items-center justify-between text-left"
                      onClick={() => setExpandedId(isOpen ? null : n.id)}
                    >
                      <div className="flex-1 pr-4">
                        <div className="font-medium">
                          {isInvite ? "Workspace invitation" : (n.type || "Notification")}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {isInvite
                            ? (d.workspace_name ? `You were invited to ${d.workspace_name}` : "You have a workspace invitation")
                            : new Date(n.createdAt).toLocaleString()}
                        </div>
                      </div>
                      {tab === "unread" ? (
                        <div className="flex items-center gap-2">
                          <Button variant="outline" size="sm" type="button" onClick={(e) => { e.stopPropagation(); onMarkRead(n); }}>Mark read</Button>
                        </div>
                      ) : null}
                    </button>
                    {isOpen && (
                      <div className="mt-3 rounded-md border p-3 text-sm">
                        {isInvite ? (
                          <div className="space-y-2">
                            <div>
                              <div>Workspace: <span className="font-medium">{d.workspace_name || d.workspace_id}</span></div>
                              <div>Role: <span className="font-medium">{d.role || "regular"}</span></div>
                            </div>
                            <div className="flex gap-2">
                              {tab === "unread" && (
                                <>
                                  <Button size="sm" onClick={() => onAcceptInvite(n)}>Accept invitation</Button>
                                  <Button variant="outline" size="sm" onClick={() => onMarkRead(n)}>Dismiss</Button>
                                </>
                              )}
                            </div>
                          </div>
                        ) : (
                          <div className="text-muted-foreground">No additional details.</div>
                        )}
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          ) : (
            <p className="text-sm text-muted-foreground">You&apos;re all caught up.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
