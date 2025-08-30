"use client";

import React, { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { changeEmail, changePassword, confirmEmail, updateProfile } from "@/lib/account";
import { toast } from "sonner";
import { User as UserIcon, Mail, Lock } from "lucide-react";
import { getMe } from "@/lib/auth";

export default function AccountPage() {
  const [loading, setLoading] = useState(true);
  const [profileName, setProfileName] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [savingProfile, setSavingProfile] = useState(false);

  const [newEmail, setNewEmail] = useState("");
  const [emailConfirmationId, setEmailConfirmationId] = useState<string | null>(null);
  const [emailCode, setEmailCode] = useState("");
  const [emailChanging, setEmailChanging] = useState(false);
  const [emailConfirming, setEmailConfirming] = useState(false);

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [passwordChanging, setPasswordChanging] = useState(false);

  // Dialog state
  const [nameOpen, setNameOpen] = useState(false);
  const [avatarOpen, setAvatarOpen] = useState(false);
  const [emailOpen, setEmailOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const me = await getMe();
        setProfileName(me.name || "");
        setNewEmail(me.email || "");
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const SUPPORT_EMAIL = "support@statgrad.app";

  async function onSaveName(e: React.FormEvent) {
    e.preventDefault();
    setSavingProfile(true);
    try {
      await updateProfile(profileName.trim(), avatarUrl.trim());
      setNameOpen(false);
      toast.success("Name updated");
    } finally {
      setSavingProfile(false);
    }
  }

  async function onSaveAvatar(e: React.FormEvent) {
    e.preventDefault();
    setSavingProfile(true);
    try {
      await updateProfile(profileName.trim(), avatarUrl.trim());
      setAvatarOpen(false);
      toast.success("Avatar updated");
    } finally {
      setSavingProfile(false);
    }
  }

  async function onStartEmailChange(e: React.FormEvent) {
    e.preventDefault();
    if (!newEmail.trim()) return;
    setEmailChanging(true);
    try {
      const { confirmation_id } = await changeEmail(newEmail.trim());
      setEmailConfirmationId(confirmation_id);
      toast.success("Verification code sent");
    } catch (e) {
      toast.error(`Could not start email change. Please try again or contact ${SUPPORT_EMAIL}.`);
    } finally {
      setEmailChanging(false);
    }
  }

  async function onConfirmEmail(e: React.FormEvent) {
    e.preventDefault();
    if (!emailConfirmationId || !emailCode.trim()) return;
    setEmailConfirming(true);
    try {
      await confirmEmail(emailConfirmationId, emailCode.trim());
      setEmailCode("");
      setEmailConfirmationId(null);
      setEmailOpen(false);
      toast.success("Email updated");
    } catch (e) {
      toast.error(`Could not confirm email. Please try again or contact ${SUPPORT_EMAIL}.`);
    } finally {
      setEmailConfirming(false);
    }
  }

  async function onChangePassword(e: React.FormEvent) {
    e.preventDefault();
    if (!currentPassword || !newPassword) return;
    setPasswordChanging(true);
    try {
      await changePassword(currentPassword, newPassword);
      setCurrentPassword("");
      setNewPassword("");
      setPasswordOpen(false);
      toast.success("Password changed");
    } catch (e) {
      toast.error(`Could not change password. Please try again or contact ${SUPPORT_EMAIL}.`);
    } finally {
      setPasswordChanging(false);
    }
  }

  if (loading) return null;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Account settings</h1>
        <p className="text-muted-foreground text-sm">Manage your profile, email, and security.</p>
      </div>

      <Tabs defaultValue="profile">
        <TabsList>
          <TabsTrigger value="profile" className="inline-flex items-center gap-2"><UserIcon className="h-4 w-4" /> Profile</TabsTrigger>
          <TabsTrigger value="email" className="inline-flex items-center gap-2"><Mail className="h-4 w-4" /> Email</TabsTrigger>
          <TabsTrigger value="security" className="inline-flex items-center gap-2"><Lock className="h-4 w-4" /> Security</TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <UserIcon className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">Display name</div>
                <div className="text-xs text-muted-foreground">Update the name shown across the app.</div>
              </div>
            </div>
            <Button size="sm" onClick={() => setNameOpen(true)}>Change</Button>
          </div>
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <UserIcon className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">Avatar URL</div>
                <div className="text-xs text-muted-foreground">Change your profile picture URL.</div>
              </div>
            </div>
            <Button size="sm" onClick={() => setAvatarOpen(true)}>Change</Button>
          </div>
        </TabsContent>

        <TabsContent value="email" className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <Mail className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">Change email</div>
                <div className="text-xs text-muted-foreground">Start email change and confirm with code.</div>
              </div>
            </div>
            <Button size="sm" onClick={() => setEmailOpen(true)}>Change</Button>
          </div>
        </TabsContent>

        <TabsContent value="security" className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <Lock className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">Change password</div>
                <div className="text-xs text-muted-foreground">Set a new account password.</div>
              </div>
            </div>
            <Button size="sm" onClick={() => setPasswordOpen(true)}>Change</Button>
          </div>
        </TabsContent>
      </Tabs>

      {/* Name Dialog */}
      <Dialog open={nameOpen} onOpenChange={setNameOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit display name</DialogTitle>
            <DialogDescription>Update your display name.</DialogDescription>
          </DialogHeader>
          <form onSubmit={onSaveName} className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="name">Name</Label>
              <Input id="name" value={profileName} onChange={(e) => setProfileName(e.target.value)} />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setNameOpen(false)}>Cancel</Button>
              <Button type="submit" disabled={savingProfile}>{savingProfile ? "Saving..." : "Save"}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Avatar Dialog */}
      <Dialog open={avatarOpen} onOpenChange={setAvatarOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Change avatar URL</DialogTitle>
            <DialogDescription>Provide a direct link to your profile image.</DialogDescription>
          </DialogHeader>
          <form onSubmit={onSaveAvatar} className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="avatar">Avatar URL</Label>
              <Input id="avatar" value={avatarUrl} onChange={(e) => setAvatarUrl(e.target.value)} placeholder="https://..." />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setAvatarOpen(false)}>Cancel</Button>
              <Button type="submit" disabled={savingProfile}>{savingProfile ? "Saving..." : "Save"}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Email Dialog */}
      <Dialog open={emailOpen} onOpenChange={(o) => { setEmailOpen(o); if (!o) { setEmailConfirmationId(null); setEmailCode(""); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Change email</DialogTitle>
            <DialogDescription>Start email change and confirm using the code sent to your new address.</DialogDescription>
          </DialogHeader>
          {!emailConfirmationId ? (
            <form onSubmit={onStartEmailChange} className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="email">New email</Label>
                <Input id="email" type="email" value={newEmail} onChange={(e) => setNewEmail(e.target.value)} />
              </div>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setEmailOpen(false)}>Cancel</Button>
                <Button type="submit" disabled={emailChanging || !newEmail.trim()}>{emailChanging ? "Sending code..." : "Send code"}</Button>
              </DialogFooter>
            </form>
          ) : (
            <form onSubmit={onConfirmEmail} className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="code">Confirmation code</Label>
                <Input id="code" value={emailCode} onChange={(e) => setEmailCode(e.target.value)} placeholder="Enter the code" />
              </div>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setEmailConfirmationId(null)}>Back</Button>
                <Button type="submit" disabled={emailConfirming || !emailCode.trim()}>{emailConfirming ? "Confirming..." : "Confirm"}</Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>

      {/* Password Dialog */}
      <Dialog open={passwordOpen} onOpenChange={setPasswordOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Change password</DialogTitle>
            <DialogDescription>Enter your current and new password.</DialogDescription>
          </DialogHeader>
          <form onSubmit={onChangePassword} className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="current">Current password</Label>
              <Input id="current" type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} />
            </div>
            <div className="space-y-1">
              <Label htmlFor="new">New password</Label>
              <Input id="new" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setPasswordOpen(false)}>Cancel</Button>
              <Button type="submit" disabled={passwordChanging || !currentPassword || !newPassword}>{passwordChanging ? "Changing..." : "Change"}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

