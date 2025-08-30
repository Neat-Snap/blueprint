"use client";

import React, { useState } from "react";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { getMe } from "@/lib/auth";
import { updateProfile, changePassword, changeEmail } from "@/lib/account";

export default function AccountPage() {
  const [profile, setProfile] = useState({ name: "", avatar_url: "" });
  const [email, setEmail] = useState("");
  const [pw, setPw] = useState({ current: "", next: "" });
  const [busy, setBusy] = useState<string | null>(null);

  async function loadMe() {
    const me = await getMe();
    setProfile({ name: me.name || "", avatar_url: "" });
    setEmail(me.email || "");
  }

  React.useEffect(() => {
    loadMe();
  }, []);

  async function onSaveProfile() {
    setBusy("profile");
    try {
      await updateProfile(profile);
    } finally {
      setBusy(null);
    }
  }

  async function onChangePassword() {
    if (!pw.current || !pw.next) return;
    setBusy("password");
    try {
      await changePassword(pw.current, pw.next);
      setPw({ current: "", next: "" });
    } finally {
      setBusy(null);
    }
  }

  async function onChangeEmail() {
    if (!email) return;
    setBusy("email");
    try {
      await changeEmail(email);
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Account</h1>
        <p className="text-muted-foreground text-sm">Manage your profile and security settings.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Profile</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input id="name" value={profile.name} onChange={(e) => setProfile((p) => ({ ...p, name: e.target.value }))} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="avatar">Avatar URL</Label>
            <Input id="avatar" value={profile.avatar_url} onChange={(e) => setProfile((p) => ({ ...p, avatar_url: e.target.value }))} placeholder="https://..." />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button onClick={onSaveProfile} disabled={busy === "profile"}>{busy === "profile" ? "Saving..." : "Save changes"}</Button>
        </CardFooter>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Email</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </CardContent>
        <CardFooter className="justify-end">
          <Button onClick={onChangeEmail} disabled={!email || busy === "email"}>{busy === "email" ? "Sending..." : "Change email"}</Button>
        </CardFooter>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Password</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="pw-current">Current password</Label>
            <Input id="pw-current" type="password" value={pw.current} onChange={(e) => setPw((p) => ({ ...p, current: e.target.value }))} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="pw-next">New password</Label>
            <Input id="pw-next" type="password" value={pw.next} onChange={(e) => setPw((p) => ({ ...p, next: e.target.value }))} />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button onClick={onChangePassword} disabled={!pw.current || !pw.next || busy === "password"}>{busy === "password" ? "Updating..." : "Change password"}</Button>
        </CardFooter>
      </Card>
    </div>
  );
}
