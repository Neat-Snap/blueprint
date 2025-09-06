"use client";

import React, { useEffect, useMemo, useRef, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { listNotifications, markNotificationRead, type Notification } from "@/lib/notifications";
import { acceptInvitation, checkInvitationStatus, type InvitationStatus } from "@/lib/teams";
import { useRouter } from "next/navigation";
import { useTeam } from "@/lib/teams-context";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Badge } from "@/components/ui/badge";

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
  const [inviteStatuses, setInviteStatuses] = useState<Record<number, InvitationStatus["status"] | "invalid" | undefined>>({});
  const markedRef = useRef<Set<number>>(new Set());

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

  // After notifications load, check status of invite notifications to filter revoked/expired/accepted
  useEffect(() => {
    const inviteNotifs = list.filter((n) => n.type === "team_invite");
    if (!inviteNotifs.length) return;
    let cancelled = false;
    (async () => {
      const results = await Promise.all(
        inviteNotifs.map(async (n) => {
          const payload = parseInviteData(n.data);
          const token = payload.token;
          if (!token) return { id: n.id, status: "invalid" as const };
          try {
            const res = await checkInvitationStatus(token);
            return { id: n.id, status: (res.status as InvitationStatus["status"]) };
          } catch (e: unknown) {
            const err = e as { response?: { status?: number } };
            // Treat not found or bad request as invalid/revoked
            if (err.response?.status === 404) {
              return { id: n.id, status: "revoked" as const };
            }
            return { id: n.id, status: "invalid" as const };
          }
        })
      );
      if (cancelled) return;
      setInviteStatuses((prev) => {
        const next = { ...prev };
        for (const r of results) next[r.id] = r.status;
        return next;
      });

      // Auto-mark non-pending invites as read to clear unread state
      for (const r of results) {
        const notif = list.find((n) => n.id === r.id);
        if (!notif) continue;
        const nonPending = r.status && r.status !== "pending";
        if (nonPending && !notif.readAt && !markedRef.current.has(r.id)) {
          markedRef.current.add(r.id);
          // Fire and forget
          markNotificationRead(r.id).then(() => {
            setList((prev) => prev.map((x) => (x.id === r.id ? { ...x, readAt: new Date().toISOString() } : x)));
          }).catch(() => {
            // ignore errors, UI still treats as read
          });
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [list]);

  async function onAcceptInvite(n: Notification) {
    const payload = parseInviteData(n.data);
    if (!payload.token) return;
    // Re-check status to avoid errors if invite was revoked/expired just now
    try {
      const status = await checkInvitationStatus(payload.token);
      if (status.status !== "pending") {
        setInviteStatuses((prev) => ({ ...prev, [n.id]: status.status as InvitationStatus["status"] }));
        return;
      }
    } catch {
      // If check fails, treat as non-pending to be safe
      setInviteStatuses((prev) => ({ ...prev, [n.id]: "revoked" }));
      return;
    }
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

  const consideredRead = (n: Notification) => {
    if (n.readAt) return true;
    if (n.type === "team_invite") {
      const st = inviteStatuses[n.id];
      // If invite is not pending (revoked/expired/accepted/invalid), treat as read
      return st && st !== "pending" ? true : false;
    }
    return false;
  };

  const unread = useMemo(() => list.filter((n) => !consideredRead(n)), [list, inviteStatuses]);
  const read = useMemo(() => list.filter((n) => consideredRead(n)), [list, inviteStatuses]);

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
                const st = isInvite ? inviteStatuses[n.id] : undefined;
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
                        <div className="flex items-center gap-2">
                          {isInvite && st && st !== "pending" && (
                            st === "accepted" ? (
                              <Badge variant="secondary">Accepted</Badge>
                            ) : st === "expired" ? (
                              <Badge variant="destructive">Expired</Badge>
                            ) : (
                              <Badge variant="destructive">Revoked</Badge>
                            )
                          )}
                          {tab === "unread" ? (
                          <div className="flex items-center gap-2">
                            {isInvite && (!st || st === "pending") && (
                              <Button size="sm" type="button" onClick={() => onAcceptInvite(n)}>Accept</Button>
                            )}
                            <Button variant="outline" size="sm" type="button" onClick={() => onMarkRead(n)}>Read</Button>
                          </div>
                          ) : null}
                        </div>
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
