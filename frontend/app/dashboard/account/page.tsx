"use client";

import React, { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { changeEmail, changePassword, confirmEmail, updateProfile } from "@/lib/account";
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

  async function onSaveProfile(e: React.FormEvent) {
    e.preventDefault();
    setSavingProfile(true);
    try {
      await updateProfile(profileName.trim(), avatarUrl.trim());
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
    } finally {
      setPasswordChanging(false);
    }
  }

  if (loading) return null;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Account settings</h1>
        <p className="text-muted-foreground text-sm">Manage your profile, email, and password.</p>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Profile</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={onSaveProfile} className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="name">Name</Label>
                <Input id="name" value={profileName} onChange={(e) => setProfileName(e.target.value)} />
              </div>
              <div className="space-y-1">
                <Label htmlFor="avatar">Avatar URL</Label>
                <Input id="avatar" value={avatarUrl} onChange={(e) => setAvatarUrl(e.target.value)} placeholder="https://..." />
              </div>
              <Button type="submit" disabled={savingProfile}>{savingProfile ? "Saving..." : "Save profile"}</Button>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Password</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={onChangePassword} className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="current">Current password</Label>
                <Input id="current" type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} />
              </div>
              <div className="space-y-1">
                <Label htmlFor="new">New password</Label>
                <Input id="new" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
              </div>
              <Button type="submit" disabled={passwordChanging || !currentPassword || !newPassword}>{passwordChanging ? "Changing..." : "Change password"}</Button>
            </form>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Email</CardTitle>
        </CardHeader>
        <CardContent>
          {!emailConfirmationId ? (
            <form onSubmit={onStartEmailChange} className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="email">New email</Label>
                <Input id="email" type="email" value={newEmail} onChange={(e) => setNewEmail(e.target.value)} />
              </div>
              <Button type="submit" disabled={emailChanging || !newEmail.trim()}>{emailChanging ? "Sending code..." : "Change email"}</Button>
            </form>
          ) : (
            <form onSubmit={onConfirmEmail} className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="code">Confirmation code</Label>
                <Input id="code" value={emailCode} onChange={(e) => setEmailCode(e.target.value)} placeholder="Enter the code sent to your email" />
              </div>
              <div className="flex gap-2">
                <Button type="submit" disabled={emailConfirming || !emailCode.trim()}>{emailConfirming ? "Confirming..." : "Confirm email"}</Button>
                <Button type="button" variant="secondary" onClick={() => setEmailConfirmationId(null)}>Back</Button>
              </div>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
