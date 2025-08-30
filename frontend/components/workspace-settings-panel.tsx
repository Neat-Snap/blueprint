"use client";

import React, { useEffect, useState } from "react";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogFooter as DialogModalFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useWorkspace } from "@/lib/workspace-context";
import { getWorkspace, updateWorkspaceName, addMember, removeMember, deleteWorkspace, createInvitation, listInvitations, revokeInvitation, updateMemberRole, type WorkspaceInvitation } from "@/lib/workspaces";
import { ALLOWED_WORKSPACE_ICONS, renderWorkspaceIcon } from "@/lib/icons";
import { getMe } from "@/lib/auth";
import { Trash2 } from "lucide-react";

export default function WorkspaceSettingsPanel() {
  const { current, refresh, setCurrentId, all } = useWorkspace();
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
  // owner reassignment removed

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

  async function handleRename() {
    if (!current || !name.trim()) return;
    setSaving(true);
    try {
      await updateWorkspaceName(current.id, name.trim(), icon.trim() || undefined);
      await refresh();
    } finally {
      setSaving(false);
    }
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
    } finally {
      setInviting(false);
    }
  }

  async function handleRevoke(invId: number) {
    if (!current) return;
    await revokeInvitation(current.id, invId);
    setInvites((prev) => prev.filter((i) => i.id !== invId));
  }

  async function handleRemoveMember(uid: number) {
    if (!current) return;
    await removeMember(current.id, uid);
    setMembers((m) => m.filter((x) => x.id !== uid));
  }

  // owner reassignment removed

  async function handleChangeRole(uid: number, role: "regular" | "admin") {
    if (!current) return;
    await updateMemberRole(current.id, uid, role);
    setMembers((prev) => prev.map((m) => (m.id === uid ? { ...m, role } : m)));
  }

  async function handleDelete() {
    if (!current) return;
    await deleteWorkspace(current.id);
    // pick another workspace if available
    await refresh();
    const remaining = all.filter((w) => w.id !== current.id);
    setCurrentId(remaining[0]?.id ?? null);
  }

  if (loading) return null;
  if (!current) return <p className="text-sm text-muted-foreground">Select a workspace from the header to manage settings.</p>;

  const myRole = meId ? members.find((m) => m.id === meId)?.role : undefined;
  const isOwner = ownerId != null && meId != null && ownerId === meId;
  const isManager = isOwner || myRole === "admin";

  return (
    <div className="space-y-6">
      {isManager && (
      <Card>
        <CardHeader>
          <CardTitle>General</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="ws-name">Name</Label>
            <Input id="ws-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Acme Corp" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="ws-icon">Icon</Label>
            <Select value={icon} onValueChange={(v) => setIcon(v)}>
              <SelectTrigger id="ws-icon" className="w-full">
                <SelectValue placeholder="Select an icon" />
              </SelectTrigger>
              <SelectContent>
                {allowedIcons.map((ic) => {
                  const label = ic.charAt(0).toUpperCase() + ic.slice(1);
                  return (
                    <SelectItem key={ic} value={ic}>
                      <span className="flex items-center gap-2">
                        {renderWorkspaceIcon(ic, "size-4")}
                        <span>{label}</span>
                      </span>
                    </SelectItem>
                  );
                })}
              </SelectContent>
            </Select>
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button onClick={handleRename} disabled={!name.trim() || saving}>{saving ? "Saving..." : "Save changes"}</Button>
        </CardFooter>
      </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Members</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-2">
            {members.length ? (
              members.map((m) => (
                <div key={m.id} className="flex items-center justify-between rounded border p-2 text-sm">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{m.name || `User #${m.id}`}</span>
                    {ownerId === m.id ? <span className="text-xs text-muted-foreground">(owner)</span> : null}
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
                      <Button size="sm" variant="destructive" onClick={() => setConfirmRemove({ open: true, userId: m.id })}>
                        <Trash2 className="mr-1 h-4 w-4" /> Remove
                      </Button>
                    </div>
                  )}
                </div>
              ))
            ) : (
              <p className="text-sm text-muted-foreground">No members.</p>
            )}
          </div>
          {isManager && (
            <div className="mt-2 grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="invite-email">Invite member by email</Label>
                <Input id="invite-email" type="email" value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} placeholder="user@example.com" />
              </div>
              <div className="space-y-2">
                <Label>Role</Label>
                <Select value={inviteRole} onValueChange={(v: "regular" | "admin") => setInviteRole(v)}>
                  <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="regular">Regular</SelectItem>
                    <SelectItem value="admin">Admin</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="sm:col-span-2 flex justify-end">
                <Button onClick={handleInvite} disabled={inviting || !inviteEmail.trim()}>{inviting ? "Sending..." : "Send invitation"}</Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {isManager && (
        <Card>
          <CardHeader>
            <CardTitle>Pending invitations</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {invites.length ? (
              <ul className="divide-y">
                {invites.map((inv) => (
                  <li key={inv.id} className="flex items-center justify-between py-2">
                    <div>
                      <div className="font-medium">{inv.email}</div>
                      <div className="text-xs text-muted-foreground">Role: {inv.role} · Expires {new Date(inv.expires_at).toLocaleDateString()}</div>
                    </div>
                    <Button size="sm" variant="outline" onClick={() => handleRevoke(inv.id)}>Revoke</Button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground">No pending invitations.</p>
            )}
          </CardContent>
        </Card>
      )}

      {isOwner && (
        <Card>
          <CardHeader>
            <CardTitle>Danger zone</CardTitle>
          </CardHeader>
          <CardContent className="flex items-center justify-between">
            <div>
              <p className="text-sm">Delete this workspace</p>
              <p className="text-xs text-muted-foreground">This action is irreversible.</p>
            </div>
            <Button variant="destructive" onClick={() => setConfirmDeleteOpen(true)}><Trash2 className="mr-1 h-4 w-4" /> Delete</Button>
          </CardContent>
        </Card>
      )}

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
