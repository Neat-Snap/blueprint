"use client";

import React, { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { changeEmail, changePassword, confirmEmail, updateProfile, getPreferences, updateTheme, updateLanguage } from "@/lib/account";
import { toast } from "sonner";
import { User as UserIcon, Mail, Lock, Settings as SettingsIcon, Sun, Moon, Languages, Laptop } from "lucide-react";
import { getMe } from "@/lib/auth";
import { useTheme } from "next-themes";
import { useRouter, usePathname } from "next/navigation";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useTranslations, useLocale } from "next-intl";

export default function AccountPage() {
  const t = useTranslations("Account");
  const [loading, setLoading] = useState(true);
  const [profileName, setProfileName] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [savingProfile, setSavingProfile] = useState(false);

  const { theme, setTheme, resolvedTheme } = useTheme();
  const [appTheme, setAppTheme] = useState<"light" | "dark" | "system">("system");
  const [language, setLanguage] = useState<string>("en");
  const router = useRouter();
  const pathname = usePathname();
  const [langUpdating, setLangUpdating] = useState(false);
  const activeLocale = useLocale();

  const [newEmail, setNewEmail] = useState("");
  const [emailConfirmationId, setEmailConfirmationId] = useState<string | null>(null);
  const [emailCode, setEmailCode] = useState("");
  const [emailChanging, setEmailChanging] = useState(false);
  const [emailConfirming, setEmailConfirming] = useState(false);

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [passwordChanging, setPasswordChanging] = useState(false);

  const [nameOpen, setNameOpen] = useState(false);
  const [avatarOpen, setAvatarOpen] = useState(false);
  const [emailOpen, setEmailOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);

  const fetchedOnceRef = useRef(false);
  useEffect(() => {
    if (fetchedOnceRef.current) return;
    fetchedOnceRef.current = true;
    (async () => {
      try {
        const me = await getMe();
        setProfileName(me.name || "");
        setNewEmail(me.email || "");
        try {
          const prefs = await getPreferences();
          const initialTheme = (prefs.theme || "system") as "light" | "dark" | "system";
          setAppTheme(initialTheme);
          setTheme(initialTheme);
          if (prefs.language) {
            setLanguage(prefs.language);
          } else if (activeLocale) {
            setLanguage(activeLocale);
          }
        } catch {
          const current = (theme || "system") as "light" | "dark" | "system";
          setAppTheme(current);
        }
      } finally {
        setLoading(false);
      }
    })();
  }, [activeLocale]);

  const SUPPORT_EMAIL = "support@statgrad.app";

  async function onSaveName(e: React.FormEvent) {
    e.preventDefault();
    setSavingProfile(true);
    try {
      await updateProfile(profileName.trim(), avatarUrl.trim());
      setNameOpen(false);
      toast.success(t("profile.nameUpdated"));
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
      toast.success(t("profile.avatarUpdated"));
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
      toast.success(t("email.codeSent"));
    } catch {
      toast.error(t("email.startFailed", { email: SUPPORT_EMAIL }));
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
      toast.success(t("email.updated"));
    } catch {
      toast.error(t("email.confirmFailed", { email: SUPPORT_EMAIL }));
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
      toast.success(t("security.passwordChanged"));
    } catch {
      toast.error(t("security.changeFailed", { email: SUPPORT_EMAIL }));
    } finally {
      setPasswordChanging(false);
    }
  }

  if (loading) return null;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("title")}</h1>
        <p className="text-muted-foreground text-sm">{t("subtitle")}</p>
      </div>

      <Tabs defaultValue="app">
        <TabsList>
          <TabsTrigger value="app" className="inline-flex items-center gap-2"><SettingsIcon className="h-4 w-4" /> {t("tabs.app")}</TabsTrigger>
          <TabsTrigger value="profile" className="inline-flex items-center gap-2"><UserIcon className="h-4 w-4" /> {t("tabs.profile")}</TabsTrigger>
          <TabsTrigger value="email" className="inline-flex items-center gap-2"><Mail className="h-4 w-4" /> {t("tabs.email")}</TabsTrigger>
          <TabsTrigger value="security" className="inline-flex items-center gap-2"><Lock className="h-4 w-4" /> {t("tabs.security")}</TabsTrigger>
        </TabsList>

        <TabsContent value="app" className="space-y-2">
          {/* Theme */}
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              {appTheme === "dark" ? (
                <Moon className="h-4 w-4 text-muted-foreground" />
              ) : appTheme === "light" ? (
                <Sun className="h-4 w-4 text-muted-foreground" />
              ) : (
                <Sun className="h-4 w-4 text-muted-foreground" />
              )}
              <div>
                <div className="text-sm font-medium">{t("theme.label")}</div>
                <div className="text-xs text-muted-foreground">
                  {appTheme === "system"
                    ? t("theme.followsSystem", { mode: resolvedTheme === "dark" ? t("theme.dark") : t("theme.light") })
                    : t("theme.using", { mode: t(`theme.${appTheme}` as any) })}
                </div>
              </div>
            </div>
            <Select
              value={appTheme}
              onValueChange={async (value) => {
                const next = value as "light" | "dark" | "system";
                setAppTheme(next);
                setTheme(next);
                try {
                  await updateTheme(next);
                  toast.success(t("theme.updated"));
                } catch {
                  toast.error(t("theme.updateFailed"));
                }
              }}
            >
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder={t("theme.placeholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="system">
                  <span className="inline-flex items-center gap-2"><Laptop className="h-4 w-4" /> {t("theme.system")}</span>
                </SelectItem>
                <SelectItem value="light">
                  <span className="inline-flex items-center gap-2"><Sun className="h-4 w-4" /> {t("theme.light")}</span>
                </SelectItem>
                <SelectItem value="dark">
                  <span className="inline-flex items-center gap-2"><Moon className="h-4 w-4" /> {t("theme.dark")}</span>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <Languages className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">{t("language.label")}</div>
                <div className="text-xs text-muted-foreground">{t("language.desc")}</div>
              </div>
            </div>
            <Select
              value={language}
              onValueChange={async (v) => {
                setLanguage(v);
                try {
                  const maxAge = 60 * 60 * 24 * 180;
                  document.cookie = `NEXT_LOCALE=${v}; Path=/; Max-Age=${maxAge}`;
                } catch {}
                try {
                  setLangUpdating(true);
                  await updateLanguage(v);
                  toast.success(t("language.updated"));
                } catch (e) {
                  toast.error(t("language.updateFailed"));
                } finally {
                  setLangUpdating(false);
                }
                if (pathname) router.replace(pathname);
                router.refresh();
              }}
            >
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder={t("language.placeholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="en">English</SelectItem>
                <SelectItem value="ru">Русский</SelectItem>
                <SelectItem value="zh">中文</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </TabsContent>

        <TabsContent value="profile" className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <UserIcon className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">{t("profile.displayName")}</div>
                <div className="text-xs text-muted-foreground">{t("profile.displayNameDesc")}</div>
              </div>
            </div>
            <Button size="sm" onClick={() => setNameOpen(true)}>{t("common.change")}</Button>
          </div>
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <UserIcon className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">{t("profile.avatarUrl")}</div>
                <div className="text-xs text-muted-foreground">{t("profile.avatarUrlDesc")}</div>
              </div>
            </div>
            <Button size="sm" onClick={() => setAvatarOpen(true)}>{t("common.change")}</Button>
          </div>
        </TabsContent>

        <TabsContent value="email" className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <Mail className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">{t("email.title")}</div>
                <div className="text-xs text-muted-foreground">{t("email.desc")}</div>
              </div>
            </div>
            <Button size="sm" onClick={() => setEmailOpen(true)}>{t("common.change")}</Button>
          </div>
        </TabsContent>

        <TabsContent value="security" className="space-y-2">
          <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
            <div className="flex items-center gap-3">
              <Lock className="h-4 w-4 text-muted-foreground" />
              <div>
                <div className="text-sm font-medium">{t("security.title")}</div>
                <div className="text-xs text-muted-foreground">{t("security.desc")}</div>
              </div>
            </div>
            <Button size="sm" onClick={() => setPasswordOpen(true)}>{t("common.change")}</Button>
          </div>
        </TabsContent>
      </Tabs>

      <Dialog open={nameOpen} onOpenChange={setNameOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("profile.editTitle")}</DialogTitle>
            <DialogDescription>{t("profile.editDesc")}</DialogDescription>
          </DialogHeader>
          <form onSubmit={onSaveName} className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="name">{t("profile.nameLabel")}</Label>
              <Input id="name" value={profileName} onChange={(e) => setProfileName(e.target.value)} />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setNameOpen(false)}>{t("common.cancel")}</Button>
              <Button type="submit" disabled={savingProfile}>{savingProfile ? t("common.saving") : t("common.save")}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={avatarOpen} onOpenChange={setAvatarOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("profile.avatarTitle")}</DialogTitle>
            <DialogDescription>{t("profile.avatarDesc")}</DialogDescription>
          </DialogHeader>
          <form onSubmit={onSaveAvatar} className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="avatar">{t("profile.avatarUrl")}</Label>
              <Input id="avatar" value={avatarUrl} onChange={(e) => setAvatarUrl(e.target.value)} placeholder="https://..." />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setAvatarOpen(false)}>{t("common.cancel")}</Button>
              <Button type="submit" disabled={savingProfile}>{savingProfile ? t("common.saving") : t("common.save")}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={emailOpen} onOpenChange={(o) => { setEmailOpen(o); if (!o) { setEmailConfirmationId(null); setEmailCode(""); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("email.dialogTitle")}</DialogTitle>
            <DialogDescription>{t("email.dialogDesc")}</DialogDescription>
          </DialogHeader>
          {!emailConfirmationId ? (
            <form onSubmit={onStartEmailChange} className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="email">{t("email.newLabel")}</Label>
                <Input id="email" type="email" value={newEmail} onChange={(e) => setNewEmail(e.target.value)} />
              </div>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setEmailOpen(false)}>{t("common.cancel")}</Button>
                <Button type="submit" disabled={emailChanging || !newEmail.trim()}>{emailChanging ? t("email.sendingCode") : t("email.sendCode")}</Button>
              </DialogFooter>
            </form>
          ) : (
            <form onSubmit={onConfirmEmail} className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="code">{t("email.codeLabel")}</Label>
                <Input id="code" value={emailCode} onChange={(e) => setEmailCode(e.target.value)} placeholder={t("email.codePlaceholder")} />
              </div>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setEmailConfirmationId(null)}>{t("common.back")}</Button>
                <Button type="submit" disabled={emailConfirming || !emailCode.trim()}>{emailConfirming ? t("email.confirming") : t("email.confirm")}</Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={passwordOpen} onOpenChange={setPasswordOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("security.dialogTitle")}</DialogTitle>
            <DialogDescription>{t("security.dialogDesc")}</DialogDescription>
          </DialogHeader>
          <form onSubmit={onChangePassword} className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="current">{t("security.current")}</Label>
              <Input id="current" type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} />
            </div>
            <div className="space-y-1">
              <Label htmlFor="new">{t("security.new")}</Label>
              <Input id="new" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setPasswordOpen(false)}>{t("common.cancel")}</Button>
              <Button type="submit" disabled={passwordChanging || !currentPassword || !newPassword}>{passwordChanging ? t("security.changing") : t("common.change")}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

