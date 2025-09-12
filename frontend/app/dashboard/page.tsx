"use client";

import React, { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useTeam } from "@/lib/teams-context";
import { getTeamOverview, type TeamOverview } from "@/lib/teams";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { listInvitations } from "@/lib/teams";
import { useTranslations } from "next-intl";

export default function DashboardPage() {
  const t = useTranslations('Dashboard');
  const { current } = useTeam();
  const [loading, setLoading] = useState(true);
  const [teamOverview, setTeamOverview] = useState<TeamOverview | null>(null);
  const [pendingInvites, setPendingInvites] = useState<number>(0);

  useEffect(() => {
    (async () => {
      if (!current) {
        setTeamOverview(null);
        setPendingInvites(0);
        setLoading(false);
        return;
      }
      try {
        const data = await getTeamOverview(current.id);
        setTeamOverview(data);
        const invites = await listInvitations(current.id);
        const now = Date.now();
        const count = invites.filter((i) => i.status === "pending" && new Date(i.expires_at).getTime() > now).length;
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
        <h1 className="text-2xl font-bold">{t('noTeamTitle')}</h1>
        <p className="text-sm text-muted-foreground">{t('noTeamDesc')}</p>
        <div>
          <Button asChild>
            <Link href="/dashboard/settings">{t('createTeam')}</Link>
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t('titleWithTeam', { team: teamOverview?.team.name || 'Team' })}</h1>
        <p className="text-muted-foreground">{t('subtitle')}</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>{t('members')}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-semibold">{teamOverview?.stats.members_count ?? 0}</div>
          </CardContent>
        </Card>
        {pendingInvites > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>{t('pendingInvitations')}</CardTitle>
            </CardHeader>
            <CardContent className="flex items-end justify-between gap-3">
              <div className="text-3xl font-semibold">{pendingInvites}</div>
              <Button asChild variant="outline">
                <Link href={`/dashboard/settings`}>{t('manage')}</Link>
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
