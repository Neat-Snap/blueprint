"use client";

import React, { useEffect, useState } from "react";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogFooter as DialogModalFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useWorkspace } from "@/lib/workspace-context";
import { getWorkspace, updateWorkspaceName, addMember, removeMember, deleteWorkspace, createInvitation, listInvitations, revokeInvitation, updateMemberRole, type WorkspaceInvitation } from "@/lib/workspaces";
import { ALLOWED_WORKSPACE_ICONS, renderWorkspaceIcon } from "@/lib/icons";
import { getMe } from "@/lib/auth";
import { Trash2, Users, Type, ShieldAlert } from "lucide-react";
import { toast } from "sonner";

export default function WorkspaceSettingsPanel() {
  const { current, refresh, setCurrentId, switchTo, all } = useWorkspace();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [name, setName] = useState("");
  const [icon, setIcon] = useState("");
  const [ownerId, setOwnerId] = useState<number | null>(null);
  const [members, setMembers] = useState<{ id: number; name: string; role: string }[]>([]);
  const [meId, setMeId] = useState<number | null>(null);
  const [newMemberId, setNewMemberId] = useState("");
  const [newMemberRole, setNewMemberRole] = useState<"regular" | "admin">("regular");
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<"regular" | "admin">("regular");
  const [inviting, setInviting] = useState(false);
  const [invites, setInvites] = useState<WorkspaceInvitation[]>([]);

  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [confirmRemove, setConfirmRemove] = useState<{ open: boolean; userId?: number }>({ open: false });

  const [renameOpen, setRenameOpen] = useState(false);
  const [iconOpen, setIconOpen] = useState(false);
  const [inviteOpen, setInviteOpen] = useState(false);

  const allowedIcons = ALLOWED_WORKSPACE_ICONS as readonly string[];

  useEffect(() => {
    (async () => {
      if (!current) {
        setLoading(false);
        return;
      }
      try {
        const [me, data] = await Promise.all([getMe(), getWorkspace(current.id)]);
        setMeId(me?.id ? Number(me.id) : null);
        setName(data.name);
        setIcon(data.icon || "");
        setOwnerId(data.owner_id);
        setMembers(data.members);
        const invs = await listInvitations(current.id);
        const now = Date.now();
        const filtered = invs.filter((i) => i.status === "pending" && new Date(i.expires_at).getTime() > now);
        setInvites(filtered);
      } finally {
        setLoading(false);
      }
    })();
  }, [current]);

  const SUPPORT_EMAIL = "support@statgrad.app";

  async function saveWorkspace(nextName: string, nextIcon?: string) {
    if (!current || !nextName.trim()) return;
    setSaving(true);
    try {
      await updateWorkspaceName(current.id, nextName.trim(), nextIcon?.trim() || undefined);
      await refresh();
      toast.success("Workspace updated");
    } catch (e) {
      toast.error(`Could not update workspace. Please try again or contact ${SUPPORT_EMAIL}.`);
    } finally {
      setSaving(false);
    }
  }

  async function handleRename() {
    await saveWorkspace(name, icon);
  }

  async function handleAddMember() {
    if (!current) return;
    const idNum = Number(newMemberId);
    if (!idNum) return;
    await addMember(current.id, idNum, newMemberRole);
    const data = await getWorkspace(current.id);
    setMembers(data.members);
    setOwnerId(data.owner_id);
    setNewMemberId("");
    setNewMemberRole("regular");
  }

  async function handleInvite() {
    if (!current || !inviteEmail.trim()) return;
    setInviting(true);
    try {
      await createInvitation(current.id, inviteEmail.trim().toLowerCase(), inviteRole);
      setInviteEmail("");
      setInviteRole("regular");
      // Optionally show a toast: invitation created and emailed.
      const invs = await listInvitations(current.id);
      const now = Date.now();
      const filtered = invs.filter((i) => i.status === "pending" && new Date(i.expires_at).getTime() > now);
      setInvites(filtered);
      setInviteOpen(false);
      toast.success("Invitation sent");
    } catch (e) {
      toast.error(`Could not send invitation. Please try again or contact ${SUPPORT_EMAIL}.`);
    } finally {
      setInviting(false);
    }
  }

  async function handleRevoke(invId: number) {
    if (!current) return;
    try {
      await revokeInvitation(current.id, invId);
      setInvites((prev) => prev.filter((i) => i.id !== invId));
      toast.success("Invitation revoked");
    } catch (e) {
      toast.error(`Could not revoke invitation. Please try again or contact ${SUPPORT_EMAIL}.`);
    }
  }

  async function handleRemoveMember(uid: number) {
    if (!current) return;
    try {
      await removeMember(current.id, uid);
      setMembers((m) => m.filter((x) => x.id !== uid));
      toast.success("Member removed");
    } catch (e) {
      toast.error(`Could not remove member. Please try again or contact ${SUPPORT_EMAIL}.`);
    }
  }

  // owner reassignment removed

  async function handleChangeRole(uid: number, role: "regular" | "admin") {
    if (!current) return;
    try {
      await updateMemberRole(current.id, uid, role);
      setMembers((prev) => prev.map((m) => (m.id === uid ? { ...m, role } : m)));
      toast.success("Role updated");
    } catch (e) {
      toast.error(`Could not update role. Please try again or contact ${SUPPORT_EMAIL}.`);
    }
  }

  async function handleDelete() {
    if (!current) return;
    try {
      await deleteWorkspace(current.id);

      await refresh();
      const remaining = all.filter((w) => w.id !== current.id);
      if (remaining[0]?.id) {
        await switchTo(remaining[0].id);
      } else {
        setCurrentId(null);
      }
      toast.success("Workspace deleted");
    } catch (e) {
      toast.error(`Could not delete workspace. Please try again or contact ${SUPPORT_EMAIL}.`);
    }
  }

  if (loading) return null;
  if (!current) return <p className="text-sm text-muted-foreground">Select a workspace from the header to manage settings.</p>;

  const myRole = meId ? members.find((m) => m.id === meId)?.role : undefined;
  const isOwner = ownerId != null && meId != null && ownerId === meId;
  const isManager = isOwner || myRole === "admin";

  return (
    <div className="space-y-6">
      <Tabs defaultValue="naming">
        <TabsList>
          <TabsTrigger value="naming" className="inline-flex items-center gap-2"><Type className="h-4 w-4" /> Naming</TabsTrigger>
          <TabsTrigger value="members" className="inline-flex items-center gap-2"><Users className="h-4 w-4" /> Members</TabsTrigger>
          <TabsTrigger value="danger" className="inline-flex items-center gap-2"><ShieldAlert className="h-4 w-4" /> Danger</TabsTrigger>
        </TabsList>

        <TabsContent value="naming" className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <Type className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">Workspace name</div>
                <div className="text-xs text-muted-foreground">Rename this workspace.</div>
              </div>
            </div>
            <Button size="sm" onClick={() => setRenameOpen(true)} disabled={!isManager}>Rename</Button>
          </div>
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              {icon ? (
                renderWorkspaceIcon(icon, "h-4 w-4")
              ) : (
                <Type className="h-4 w-4 text-muted-foreground" />
              )}
              <div className="text-sm font-medium">Workspace icon</div>
            </div>
            <Button size="sm" onClick={() => setIconOpen(true)} disabled={!isManager}>Change</Button>
          </div>
        </TabsContent>

        <TabsContent value="members" className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 text-base font-semibold"><Users className="h-4 w-4 text-muted-foreground" /> Members and pending invitations</div>
            {isManager && (
              <Button size="default" onClick={() => setInviteOpen(true)}>Invite</Button>
            )}
          </div>
          <div className="space-y-2">
            {members.length ? (
              members.map((m) => (
                <div key={m.id} className="flex items-center justify-between rounded-md border p-2 text-sm transition-colors hover:bg-muted/40">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{m.name || `User #${m.id}`}</span>
                    {ownerId === m.id ? <Badge variant="secondary" className="px-1.5 py-0 h-5">Owner</Badge> : null}
                  </div>
                  {isManager && (
                    <div className="flex items-center gap-2">
                      {isOwner && ownerId !== m.id && (
                        <Select value={(m.role as "regular" | "admin") || "regular"} onValueChange={(v: "regular" | "admin") => handleChangeRole(m.id, v)}>
                          <SelectTrigger className="w-[130px]"><SelectValue /></SelectTrigger>
                          <SelectContent>
                            <SelectItem value="regular">Regular</SelectItem>
                            <SelectItem value="admin">Admin</SelectItem>
                          </SelectContent>
                        </Select>
                      )}
                      {ownerId !== m.id && (
                        <Button size="sm" variant="destructive" onClick={() => setConfirmRemove({ open: true, userId: m.id })}>
                          <Trash2 className="mr-1 h-4 w-4" /> Remove
                        </Button>
                      )}
                    </div>
                  )}
                </div>
              ))
            ) : (
              <p className="text-sm text-muted-foreground">No members.</p>
            )}
          </div>
          {isManager && (
            <div className="space-y-2 pt-2">
              <div className="text-sm font-medium text-muted-foreground">Pending invitations</div>
              {invites.length ? (
                <ul className="divide-y rounded-md border">
                  {invites.map((inv) => (
                    <li key={inv.id} className="flex items-center justify-between p-2 text-sm transition-colors hover:bg-muted/40">
                      <div className="flex items-center gap-2">
                        <div className="font-medium">{inv.email}</div>
                        <Badge variant="outline" className="px-1.5 py-0 h-5 capitalize">{inv.role}</Badge>
                        <span className="text-xs text-muted-foreground">Expires {new Date(inv.expires_at).toLocaleDateString()}</span>
                      </div>
                      <Button size="sm" variant="outline" onClick={() => handleRevoke(inv.id)}>Revoke</Button>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-sm text-muted-foreground">No pending invitations.</p>
              )}
            </div>
          )}
        </TabsContent>

        <TabsContent value="danger" className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div>
              <div className="text-sm font-medium">Delete workspace</div>
              <div className="text-xs text-muted-foreground">This action cannot be undone.</div>
            </div>
            <Button size="sm" variant="destructive" onClick={() => setConfirmDeleteOpen(true)} disabled={!isOwner}><Trash2 className="mr-1 h-4 w-4" /> Delete</Button>
          </div>
        </TabsContent>
      </Tabs>

      {/* Rename Dialog */}
      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename workspace</DialogTitle>
            <DialogDescription>Update the workspace name.</DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="ws-name">Name</Label>
              <Input id="ws-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Acme Corp" />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setRenameOpen(false)}>Cancel</Button>
              <Button onClick={async () => { await handleRename(); setRenameOpen(false); }} disabled={!name.trim() || saving}>{saving ? "Saving..." : "Save"}</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Icon Picker Dialog */}
      <Dialog open={iconOpen} onOpenChange={setIconOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Select icon</DialogTitle>
            <DialogDescription>Choose an icon for this workspace.</DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="grid grid-cols-6 gap-2 sm:grid-cols-8">
              {allowedIcons.map((ic) => (
                <button
                  key={ic}
                  type="button"
                  onClick={async () => { setIcon(ic); await saveWorkspace(name, ic); setIconOpen(false); }}
                  className={`flex h-10 w-10 items-center justify-center rounded border transition-colors ${icon === ic ? "border-ring bg-accent" : "hover:bg-muted"}`}
                  aria-label={ic}
                >
                  {renderWorkspaceIcon(ic, "size-5")}
                </button>
              ))}
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Invitations Dialog */}
      <Dialog open={inviteOpen} onOpenChange={(o) => { setInviteOpen(o); if (!o) { setInviteEmail(""); setInviteRole("regular"); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Invite member</DialogTitle>
            <DialogDescription>Send an invitation by email.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {isManager && (
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="invite-email">Invite by email</Label>
                  <Input id="invite-email" type="email" value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} placeholder="user@example.com" />
                </div>
                <div className="space-y-2 sm:col-span-2">
                  <Label>Role</Label>
                  <RadioGroup value={inviteRole} onValueChange={(v) => setInviteRole(v as "regular" | "admin")}>
                    <label className="flex items-start gap-3 rounded-md border p-2 transition-colors hover:bg-muted/50">
                      <RadioGroupItem value="regular" />
                      <div>
                        <div className="text-sm font-medium">Regular</div>
                        <div className="text-xs text-muted-foreground">Can view and edit workspace content. Cannot manage members or delete workspace.</div>
                      </div>
                    </label>
                    <label className="flex items-start gap-3 rounded-md border p-2 transition-colors hover:bg-muted/50">
                      <RadioGroupItem value="admin" />
                      <div>
                        <div className="text-sm font-medium">Admin</div>
                        <div className="text-xs text-muted-foreground">Can manage members and invitations, and edit workspace settings.</div>
                      </div>
                    </label>
                  </RadioGroup>
                </div>
                <div className="sm:col-span-2 flex justify-end gap-2">
                  <Button variant="outline" onClick={() => setInviteOpen(false)}>Cancel</Button>
                  <Button onClick={handleInvite} disabled={inviting || !inviteEmail.trim()}>{inviting ? "Sending..." : "Send invitation"}</Button>
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Confirm Delete Dialog */}
      <Dialog open={confirmDeleteOpen} onOpenChange={setConfirmDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete workspace?</DialogTitle>
            <DialogDescription>This action cannot be undone. All data for this workspace will be permanently removed.</DialogDescription>
          </DialogHeader>
          <DialogModalFooter>
            <Button variant="outline" onClick={() => setConfirmDeleteOpen(false)}>Cancel</Button>
            <Button variant="destructive" onClick={async () => { setConfirmDeleteOpen(false); await handleDelete(); }}>Delete</Button>
          </DialogModalFooter>
        </DialogContent>
      </Dialog>

      {/* Confirm Remove Member */}
      <Dialog open={confirmRemove.open} onOpenChange={(o) => setConfirmRemove({ open: o })}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove member?</DialogTitle>
            <DialogDescription>This user will lose access to this workspace.</DialogDescription>
          </DialogHeader>
          <DialogModalFooter>
            <Button variant="outline" onClick={() => setConfirmRemove({ open: false })}>Cancel</Button>
            <Button variant="destructive" onClick={async () => { const uid = confirmRemove.userId!; setConfirmRemove({ open: false }); await handleRemoveMember(uid); }}>Remove</Button>
          </DialogModalFooter>
        </DialogContent>
      </Dialog>

      {/* owner reassignment dialog removed */}
    </div>
  );
}
