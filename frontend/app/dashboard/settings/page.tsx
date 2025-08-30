"use client";

import React from "react";
import WorkspaceSettingsPanel from "@/components/workspace-settings-panel";

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Workspace settings</h1>
        <p className="text-muted-foreground text-sm">Manage the current workspace name, members, roles, and deletion.</p>
      </div>
      <WorkspaceSettingsPanel />
    </div>
  );
}
