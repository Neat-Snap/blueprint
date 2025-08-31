"use client";

import React, { useEffect, useMemo, useState } from "react";
import { ChevronsUpDown, Plus } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuShortcut, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar";
import { useTeam } from "@/lib/teams-context";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ALLOWED_TEAM_ICONS, renderTeamIcon } from "@/lib/icons";
import { toast } from "sonner";

export function TeamSwitcher() {
  const { current, all, switchTo, createTeam } = useTeam();
  const { isMobile } = useSidebar();
  const [openCreate, setOpenCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [icon, setIcon] = useState("");
  const [iconOpen, setIconOpen] = useState(false);

  const currentBadge = useMemo(() => {
    if (current?.icon && current.icon.trim()) return current.icon.trim();
    const n = current?.name?.trim() || "?";
    return n.slice(0, 2).toUpperCase();
  }, [current]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;
      const num = Number(e.key);
      if (Number.isFinite(num) && num >= 1 && num <= 9) {
        const idx = num - 1;
        if (all[idx]) {
          e.preventDefault();

          switchTo(all[idx].id);
        }
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [all, switchTo]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setCreating(true);
    try {
      await createTeam(name.trim(), icon.trim() || undefined);
      setOpenCreate(false);
      setName("");
      setIcon("");
      toast.success("Team created");
    } catch {
      toast.error("Could not create team. Please try again or contact support@statgrad.app.");
    } finally {
      setCreating(false);
    }
  }

  return (
    <>
      <SidebarMenu>
        <SidebarMenuItem>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <SidebarMenuButton
                size="lg"
                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
              >
                <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg">
                  {current?.icon ? (
                    renderTeamIcon(current.icon, "size-4") || <span className="text-xs font-medium">{currentBadge}</span>
                  ) : (
                    <span className="text-xs font-medium">{currentBadge}</span>
                  )}
                </div>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">{current?.name || "Select team"}</span>
                  <span className="truncate text-xs">{all.length === 1 ? "Free" : `${all.length} teams`}</span>
                </div>
                <ChevronsUpDown className="ml-auto" />
              </SidebarMenuButton>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
              align="start"
              side={isMobile ? "bottom" : "right"}
              sideOffset={4}
            >
              <DropdownMenuLabel className="text-muted-foreground text-xs">Teams</DropdownMenuLabel>
              {all.map((w: { id: number; name: string; icon?: string }, index: number) => {
                const badge = (w.icon && w.icon.trim()) ? w.icon.trim() : (w.name?.slice(0, 2).toUpperCase());
                return (
                  <DropdownMenuItem key={w.id} onClick={() => switchTo(w.id)} className="gap-2 p-2">
                    <div className="flex size-6 items-center justify-center rounded-md border">
                      {w.icon ? (renderTeamIcon(w.icon, "size-4") || <span className="text-[11px] font-medium">{badge}</span>) : (
                        <span className="text-[11px] font-medium">{badge}</span>
                      )}
                    </div>
                    {w.name}
                    {index < 9 && <DropdownMenuShortcut>⌘{index + 1}</DropdownMenuShortcut>}
                  </DropdownMenuItem>
                );
              })}
              <DropdownMenuSeparator />
              <DropdownMenuItem className="gap-2 p-2" onClick={() => setOpenCreate(true)}>
                <div className="flex size-6 items-center justify-center rounded-md border bg-transparent">
                  <Plus className="size-4" />
                </div>
                <div className="text-muted-foreground font-medium">Add team</div>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>

      {/* Create Team Dialog */}
      <Dialog open={openCreate} onOpenChange={setOpenCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create team</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setIconOpen(true)}
                  className={`flex h-9 w-9 items-center justify-center rounded-md border transition-colors ${icon ? "border-ring bg-accent/40" : "hover:bg-muted"}`}
                  aria-label="Choose icon"
                >
                  {icon ? (
                    renderTeamIcon(icon, "size-4")
                  ) : (
                    <span className="text-[11px] font-medium">
                      {(name.trim() ? name.trim().slice(0, 2) : "TM").toUpperCase()}
                    </span>
                  )}
                </button>
                <Input id="name" value={name} onChange={(e) => setName(e.target.value)} placeholder="My team" />
              </div>
            </div>
            <DialogFooter>
              <Button type="submit" disabled={creating}>{creating ? "Creating..." : "Create"}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Icon Picker for Create */}
      <Dialog open={iconOpen} onOpenChange={setIconOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Select icon</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div className="grid grid-cols-6 gap-2 sm:grid-cols-8">
              {ALLOWED_TEAM_ICONS.map((ic: string) => (
                <button
                  key={ic}
                  type="button"
                  onClick={() => { setIcon(ic); setIconOpen(false); }}
                  className={`flex h-10 w-10 items-center justify-center rounded border transition-colors ${icon === ic ? "border-ring bg-accent" : "hover:bg-muted"}`}
                  aria-label={ic}
                >
                  {renderTeamIcon(ic, "size-5")}
                </button>
              ))}
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
