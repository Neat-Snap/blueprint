"use client";

import React, { useEffect, useRef, useState } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogFooter as DialogModalFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useTeam } from "@/lib/teams-context";
import { getTeam, updateTeam, removeMember, deleteTeam, createInvitation, listInvitations, revokeInvitation, updateMemberRole, type TeamInvitation } from "@/lib/teams";
import { ALLOWED_TEAM_ICONS, renderTeamIcon } from "@/lib/icons";
import { getMe } from "@/lib/auth";
import { Trash2, Users, Type, ShieldAlert } from "lucide-react";
import { toast } from "sonner";
import { useTranslations } from "next-intl";

export default function TeamSettingsPanel() {
  const t = useTranslations('TeamSettings');
  const { current, refresh, setCurrentId, switchTo, all } = useTeam();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [name, setName] = useState("");
  const [icon, setIcon] = useState("");
  const [ownerId, setOwnerId] = useState<number | null>(null);
  const [members, setMembers] = useState<{ id: number; name: string; email: string; role: string }[]>([]);
  const [meId, setMeId] = useState<number | null>(null);
  
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<"regular" | "admin">("regular");
  const [inviting, setInviting] = useState(false);
  const [invites, setInvites] = useState<TeamInvitation[]>([]);

  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [confirmRemove, setConfirmRemove] = useState<{ open: boolean; userId?: number }>({ open: false });

  const [renameOpen, setRenameOpen] = useState(false);
  const [iconOpen, setIconOpen] = useState(false);
  const [inviteOpen, setInviteOpen] = useState(false);

  const allowedIcons = ALLOWED_TEAM_ICONS as readonly string[];

  const lastLoadedTeamId = useRef<number | null>(null);
  useEffect(() => {
    (async () => {
      if (!current) {
        setLoading(false);
        lastLoadedTeamId.current = null;
        return;
      }

      if (lastLoadedTeamId.current === current.id) return;
      lastLoadedTeamId.current = current.id;
      try {
        const [me, data] = await Promise.all([getMe(), getTeam(current.id)]);
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

  async function saveTeam(nextName: string, nextIcon?: string) {
    if (!current || !nextName.trim()) return;
    setSaving(true);
    try {
      await updateTeam(current.id, nextName.trim(), nextIcon?.trim() || undefined);
      await refresh();
      toast.success(t('toast.teamUpdated'));
    } catch {
      toast.error(t('toast.teamUpdateFailed', { email: SUPPORT_EMAIL }));
    } finally {
      setSaving(false);
    }
  }

  async function handleRename() {
    await saveTeam(name, icon);
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
      toast.success(t('toast.invitationSent'));
    } catch {
      toast.error(t('toast.invitationFailed', { email: SUPPORT_EMAIL }));
    } finally {
      setInviting(false);
    }
  }

  async function handleRevoke(invId: number) {
    if (!current) return;
    try {
      await revokeInvitation(current.id, invId);
      setInvites((prev) => prev.filter((i) => i.id !== invId));
      toast.success(t('toast.invitationRevoked'));
    } catch {
      toast.error(t('toast.invitationRevokeFailed', { email: SUPPORT_EMAIL }));
    }
  }

  async function handleRemoveMember(uid: number) {
    if (!current) return;
    try {
      await removeMember(current.id, uid);
      setMembers((m) => m.filter((x) => x.id !== uid));
      toast.success(t('toast.memberRemoved'));
    } catch {
      toast.error(t('toast.memberRemoveFailed', { email: SUPPORT_EMAIL }));
    }
  }

  async function handleChangeRole(uid: number, role: "regular" | "admin") {
    if (!current) return;
    try {
      await updateMemberRole(current.id, uid, role);
      setMembers((prev) => prev.map((m) => (m.id === uid ? { ...m, role } : m)));
      toast.success(t('toast.roleUpdated'));
    } catch {
      toast.error(t('toast.roleUpdateFailed', { email: SUPPORT_EMAIL }));
    }
  }

  async function handleDelete() {
    if (!current) return;
    try {
      await deleteTeam(current.id);

      await refresh();
      const remaining = all.filter((w: { id: number; name: string; icon?: string }) => w.id !== current.id);
      if (remaining[0]?.id) {
        await switchTo(remaining[0].id);
      } else {
        setCurrentId(null);
      }
      toast.success(t('toast.teamDeleted'));
    } catch {
      toast.error(t('toast.teamDeleteFailed', { email: SUPPORT_EMAIL }));
    }
  }

  if (loading) return null;
  if (!current) return <p className="text-sm text-muted-foreground">{t('noTeamSelected')}</p>;

  const myRole = meId ? members.find((m) => m.id === meId)?.role : undefined;
  const isOwner = ownerId != null && meId != null && ownerId === meId;
  const isManager = isOwner || myRole === "admin";

  return (
    <div className="space-y-6">
      <Tabs defaultValue="naming">
        <TabsList>
          <TabsTrigger value="naming" className="inline-flex items-center gap-2"><Type className="h-4 w-4" /> {t('tabs.naming')}</TabsTrigger>
          <TabsTrigger value="members" className="inline-flex items-center gap-2"><Users className="h-4 w-4" /> {t('tabs.members')}</TabsTrigger>
          <TabsTrigger value="danger" className="inline-flex items-center gap-2"><ShieldAlert className="h-4 w-4" /> {t('tabs.danger')}</TabsTrigger>
        </TabsList>

        <TabsContent value="naming" className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <Type className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">{t('naming.teamName')}</div>
                <div className="text-xs text-muted-foreground">{t('naming.teamNameDesc')}</div>
              </div>
            </div>
            <Button size="sm" onClick={() => setRenameOpen(true)} disabled={!isManager}>{t('common.rename')}</Button>
          </div>
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              {icon ? (
                renderTeamIcon(icon, "h-4 w-4")
              ) : (
                <Type className="h-4 w-4 text-muted-foreground" />
              )}
              <div className="text-sm font-medium">{t('naming.teamIcon')}</div>
            </div>
            <Button size="sm" onClick={() => setIconOpen(true)} disabled={!isManager}>{t('common.change')}</Button>
          </div>
        </TabsContent>

        <TabsContent value="members" className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 text-base font-semibold"><Users className="h-4 w-4 text-muted-foreground" /> {t('members.title')}</div>
            {isManager && (
              <Button size="default" onClick={() => setInviteOpen(true)}>{t('members.invite')}</Button>
            )}
          </div>
          <div className="space-y-2">
            {members.length ? (
              members.map((m) => (
                <div key={m.id} className="flex items-center justify-between rounded-md border p-2 text-sm transition-colors hover:bg-muted/40">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{m.name || m.email || `${t('members.user')} #${m.id}`}</span>
                    {ownerId === m.id ? <Badge variant="secondary" className="px-1.5 py-0 h-5">{t('members.owner')}</Badge> : null}
                  </div>
                  {isManager && (
                    <div className="flex items-center gap-2">
                      {isOwner && ownerId !== m.id && (
                        <Select value={(m.role as "regular" | "admin") || "regular"} onValueChange={(v: "regular" | "admin") => handleChangeRole(m.id, v)}>
                          <SelectTrigger className="w-[130px]"><SelectValue /></SelectTrigger>
                          <SelectContent>
                            <SelectItem value="regular">{t('members.roleRegular')}</SelectItem>
                            <SelectItem value="admin">{t('members.roleAdmin')}</SelectItem>
                          </SelectContent>
                        </Select>
                      )}
                      {ownerId !== m.id && (
                        <Button size="sm" variant="destructive" onClick={() => setConfirmRemove({ open: true, userId: m.id })}>
                          <Trash2 className="mr-1 h-4 w-4" /> {t('members.remove')}
                        </Button>
                      )}
                    </div>
                  )}
                </div>
              ))
            ) : (
              <p className="text-sm text-muted-foreground">{t('members.none')}</p>
            )}
          </div>
          {isManager && (
            <div className="space-y-2 pt-2">
              <div className="text-sm font-medium text-muted-foreground">{t('members.pendingInvitations')}</div>
              {invites.length ? (
                <ul className="divide-y rounded-md border">
                  {invites.map((inv) => (
                    <li key={inv.id} className="flex items-center justify-between p-2 text-sm transition-colors hover:bg-muted/40">
                      <div className="flex items-center gap-2">
                        <div className="font-medium">{inv.email}</div>
                        <Badge variant="outline" className="px-1.5 py-0 h-5 capitalize">{inv.role}</Badge>
                        <span className="text-xs text-muted-foreground">{t('members.expires', { date: new Date(inv.expires_at).toLocaleDateString() })}</span>
                      </div>
                      <Button size="sm" variant="outline" onClick={() => handleRevoke(inv.id)}>{t('members.revoke')}</Button>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-sm text-muted-foreground">{t('members.noPending')}</p>
              )}
            </div>
          )}
        </TabsContent>

        <TabsContent value="danger" className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div>
              <div className="text-sm font-medium">{t('danger.deleteTitle')}</div>
              <div className="text-xs text-muted-foreground">{t('danger.deleteDesc')}</div>
            </div>
            <Button size="sm" variant="destructive" onClick={() => setConfirmDeleteOpen(true)} disabled={!isOwner}><Trash2 className="mr-1 h-4 w-4" /> {t('common.delete')}</Button>
          </div>
        </TabsContent>
      </Tabs>

      {/* Rename Dialog */}
      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('dialogs.rename.title')}</DialogTitle>
            <DialogDescription>{t('dialogs.rename.desc')}</DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="ws-name">{t('dialogs.rename.nameLabel')}</Label>
              <Input id="ws-name" value={name} onChange={(e) => setName(e.target.value)} placeholder={t('dialogs.rename.namePlaceholder')} />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setRenameOpen(false)}>{t('common.cancel')}</Button>
              <Button onClick={async () => { await handleRename(); setRenameOpen(false); }} disabled={!name.trim() || saving}>{saving ? t('common.saving') : t('common.save')}</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Icon Picker Dialog */}
      <Dialog open={iconOpen} onOpenChange={setIconOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('dialogs.icon.title')}</DialogTitle>
            <DialogDescription>{t('dialogs.icon.desc')}</DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="grid grid-cols-6 gap-2 sm:grid-cols-8">
              {allowedIcons.map((ic) => (
                <button
                  key={ic}
                  type="button"
                  onClick={async () => { setIcon(ic); await saveTeam(name, ic); setIconOpen(false); }}
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

      {/* Invitations Dialog */}
      <Dialog open={inviteOpen} onOpenChange={(o) => { setInviteOpen(o); if (!o) { setInviteEmail(""); setInviteRole("regular"); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('dialogs.invite.title')}</DialogTitle>
            <DialogDescription>{t('dialogs.invite.desc')}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {isManager && (
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="invite-email">{t('dialogs.invite.emailLabel')}</Label>
                  <Input id="invite-email" type="email" value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} placeholder="user@example.com" />
                </div>
                <div className="space-y-2 sm:col-span-2">
                  <Label>{t('dialogs.invite.roleLabel')}</Label>
                  <RadioGroup value={inviteRole} onValueChange={(v) => setInviteRole(v as "regular" | "admin")}>
                    <label className="flex items-start gap-3 rounded-md border p-2 transition-colors hover:bg-muted/50">
                      <RadioGroupItem value="regular" />
                      <div>
                        <div className="text-sm font-medium">{t('dialogs.invite.roleRegular')}</div>
                        <div className="text-xs text-muted-foreground">{t('dialogs.invite.roleRegularDesc')}</div>
                      </div>
                    </label>
                    <label className="flex items-start gap-3 rounded-md border p-2 transition-colors hover:bg-muted/50">
                      <RadioGroupItem value="admin" />
                      <div>
                        <div className="text-sm font-medium">{t('dialogs.invite.roleAdmin')}</div>
                        <div className="text-xs text-muted-foreground">{t('dialogs.invite.roleAdminDesc')}</div>
                      </div>
                    </label>
                  </RadioGroup>
                </div>
                <div className="sm:col-span-2 flex justify-end gap-2">
                  <Button variant="outline" onClick={() => setInviteOpen(false)}>{t('common.cancel')}</Button>
                  <Button onClick={handleInvite} disabled={inviting || !inviteEmail.trim()}>{inviting ? t('common.sending') : t('dialogs.invite.send')}</Button>
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
            <DialogTitle>{t('dialogs.delete.title')}</DialogTitle>
            <DialogDescription>{t('dialogs.delete.desc')}</DialogDescription>
          </DialogHeader>
          <DialogModalFooter>
            <Button variant="outline" onClick={() => setConfirmDeleteOpen(false)}>{t('common.cancel')}</Button>
            <Button variant="destructive" onClick={async () => { setConfirmDeleteOpen(false); await handleDelete(); }}>{t('common.delete')}</Button>
          </DialogModalFooter>
        </DialogContent>
      </Dialog>

      {/* Confirm Remove Member */}
      <Dialog open={confirmRemove.open} onOpenChange={(o) => setConfirmRemove({ open: o })}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('dialogs.remove.title')}</DialogTitle>
            <DialogDescription>{t('dialogs.remove.desc')}</DialogDescription>
          </DialogHeader>
          <DialogModalFooter>
            <Button variant="outline" onClick={() => setConfirmRemove({ open: false })}>{t('common.cancel')}</Button>
            <Button variant="destructive" onClick={async () => { const uid = confirmRemove.userId!; setConfirmRemove({ open: false }); await handleRemoveMember(uid); }}>{t('members.remove')}</Button>
          </DialogModalFooter>
        </DialogContent>
      </Dialog>

      {/* owner reassignment dialog removed */}
    </div>
  );
}
