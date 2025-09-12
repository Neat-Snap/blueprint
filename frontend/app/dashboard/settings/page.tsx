"use client";

import React from "react";
import TeamSettingsPanel from "@/components/team-settings-panel";
import { useTranslations } from "next-intl";

export default function SettingsPage() {
  const t = useTranslations('SettingsPage');
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t('title')}</h1>
        <p className="text-muted-foreground text-sm">{t('subtitle')}</p>
      </div>
      <TeamSettingsPanel />
    </div>
  );
}
