"use client";

import React, { useState } from "react";
import { ChevronsUpDown, Plus } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import { useWorkspace } from "@/lib/workspace-context";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export function WorkspaceSwitcher() {
  const { current, all, setCurrentId, createWorkspace } = useWorkspace();
  const [openCreate, setOpenCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setCreating(true);
    try {
      await createWorkspace(name.trim());
      setOpenCreate(false);
      setName("");
    } finally {
      setCreating(false);
    }
  }

  // No inline edit; use Settings page

  return (
    <>
      <SidebarMenu>
        <SidebarMenuItem>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <SidebarMenuButton size="lg" className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground">
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">{current?.name || "Select workspace"}</span>
                  <span className="truncate text-xs text-muted-foreground">Switch workspace</span>
                </div>
                <ChevronsUpDown className="ml-auto size-4" />
              </SidebarMenuButton>
            </DropdownMenuTrigger>
            <DropdownMenuContent side="bottom" align="start" sideOffset={4} className="w-(--radix-dropdown-menu-trigger-width)">
              <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
              <DropdownMenuSeparator />
              {all.map((w) => (
                <DropdownMenuItem key={w.id} onClick={() => setCurrentId(w.id)}>
                  {w.name}
                </DropdownMenuItem>
              ))}
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => setOpenCreate(true)}>
                <Plus className="mr-2 h-4 w-4" /> New workspace
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>

      {/* Create Workspace Dialog */}
      <Dialog open={openCreate} onOpenChange={setOpenCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New workspace</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="ws-name">Name</Label>
              <Input id="ws-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Acme Corp" />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpenCreate(false)}>Cancel</Button>
              <Button type="submit" disabled={!name.trim() || creating}>{creating ? "Creating..." : "Create"}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
