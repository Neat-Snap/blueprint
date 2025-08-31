"use client";

import React from "react";
import TeamSettingsPanel from "@/components/team-settings-panel";

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Team settings</h1>
        <p className="text-muted-foreground text-sm">Manage the current team name, members, roles, and deletion.</p>
      </div>
      <TeamSettingsPanel />
    </div>
  );
}
