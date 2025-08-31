"use client";

import React, { useEffect, useMemo, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { listNotifications, markNotificationRead, type Notification } from "@/lib/notifications";
import { acceptInvitation } from "@/lib/teams";
import { useRouter } from "next/navigation";
import { useTeam } from "@/lib/teams-context";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";

function parseInviteData(data: string): { team_id?: number; team_name?: string; token?: string; role?: string } {
  try {
    return JSON.parse(data || "{}");
  } catch {
    return {} as { team_id?: number; team_name?: string; token?: string; role?: string };
  }
}

export default function NotificationsPage() {
  const router = useRouter();
  const { switchTo } = useTeam();
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
    if (payload.team_id) {
      await switchTo(Number(payload.team_id));
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
            <Tabs value={tab} onValueChange={(v) => setTab(v as "unread" | "read")}
              className="text-sm">
              <TabsList>
                <TabsTrigger value="unread">Unread ({unread.length})</TabsTrigger>
                <TabsTrigger value="read">Read ({read.length})</TabsTrigger>
              </TabsList>
            </Tabs>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {(tab === "unread" ? unread : read).length ? (
            <ul className="divide-y">
              {(tab === "unread" ? unread : read).map((n) => {
                const isInvite = n.type === "team_invite";
                const isOpen = expandedId === n.id;
                const d = isInvite ? parseInviteData(n.data) : {};
                return (
                  <li key={n.id} className="py-3">
                    <Collapsible open={isOpen} onOpenChange={(o) => setExpandedId(o ? n.id : null)}>
                      <div className="flex items-start justify-between gap-3">
                        <CollapsibleTrigger asChild>
                          <button className="flex-1 text-left">
                            <div className="font-medium">{isInvite ? "Team invitation" : (n.type || "Notification")}</div>
                            <div className="text-xs text-muted-foreground">
                              {isInvite ? (d.team_name ? `You were invited to ${d.team_name}` : "You have a team invitation") : new Date(n.createdAt).toLocaleString()}
                            </div>
                          </button>
                        </CollapsibleTrigger>
                        {tab === "unread" ? (
                          <div className="flex items-center gap-2">
                            {isInvite && (
                              <Button size="sm" type="button" onClick={() => onAcceptInvite(n)}>Accept</Button>
                            )}
                            <Button variant="outline" size="sm" type="button" onClick={() => onMarkRead(n)}>Read</Button>
                          </div>
                        ) : null}
                      </div>
                      <CollapsibleContent>
                        <div className="mt-3 rounded-md border p-3 text-sm">
                          {isInvite ? (
                            <div className="space-y-2">
                              <div>
                                <div>Team: <span className="font-medium">{d.team_name || d.team_id}</span></div>
                                <div>Role: <span className="font-medium">{d.role || "regular"}</span></div>
                              </div>
                            </div>
                          ) : (
                            <div className="text-muted-foreground">No additional details.</div>
                          )}
                        </div>
                      </CollapsibleContent>
                    </Collapsible>
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
