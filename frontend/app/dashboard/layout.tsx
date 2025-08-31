"use client";

import React, { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { LayoutDashboard, Settings2, MessageSquareText, HelpCircle } from "lucide-react";

import { AppSidebar, type NavMainItem, type ProjectItem, type SecondaryItem } from "@/components/app-sidebar";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { getMe } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { TeamProvider, useTeam } from "@/lib/teams-context";
import { TeamSwitcher } from "@/components/team-switcher";
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent } from "@/components/ui/dropdown-menu";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { LoadingScreen } from "@/components/loading-screen";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<{ name: string; email: string; avatar: string }>({ name: "", email: "", avatar: "" });
  const [authChecked, setAuthChecked] = useState(false);
  const [sending, setSending] = useState(false);
  const [feedback, setFeedback] = useState("");
  const [feedbackOpen, setFeedbackOpen] = useState(false);
  const fetchedAuthRef = useRef(false);

  useEffect(() => {
    if (fetchedAuthRef.current) return;
    fetchedAuthRef.current = true;
    (async () => {
      try {
        const me = await getMe();
        setUser({ name: me.name || "", email: me.email || "", avatar: "" });
      } catch {
        router.replace("/auth/login");
        return;
      } finally {
        setAuthChecked(true);
      }
    })();
  }, [router]);

  const navMain: NavMainItem[] = [
    { title: "Dashboard", url: "/dashboard", icon: LayoutDashboard, isActive: pathname === "/dashboard" },
    { title: "Settings", url: "/dashboard/settings", icon: Settings2, isActive: pathname?.startsWith("/dashboard/settings") },
  ];

  const projects: ProjectItem[] = [];
  const navSecondary: SecondaryItem[] = [];

  if (!authChecked) {
    return <LoadingScreen label="Loading Dashboard" />;
  }

  async function submitFeedback() {
    if (!feedback.trim()) return;
    setSending(true);
    try {
      const res = await fetch("/api/feedback", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: feedback }),
      });
      if (!res.ok) {
        if (res.status === 429) {
          toast.error("Daily feedback limit reached. Please try again tomorrow.");
        } else {
          toast.error("Could not send feedback. Please try again later or contact support@statgrad.app.");
        }
        return;
      }
      setFeedback("");
      setFeedbackOpen(false);
      toast.success("Thanks for your feedback!");
    } catch {
      toast.error("Could not send feedback. Please try again later or contact support@statgrad.app.");
    } finally {
      setSending(false);
    }
  }

  return (
    <TeamProvider>
      <SidebarProvider>
        <TeamSwitchOverlay />
        <AppSidebar
          org={{ name: "Your App", plan: "Free", href: "/dashboard" }}
          user={{ name: user.name || user.email || "User", email: user.email || "", avatar: user.avatar }}
          navMain={navMain}
          projects={projects}
          navSecondary={navSecondary}
          headerSlot={<TeamSwitcher />}
        />
        <SidebarInset>
          <div className="rounded-t-lg overflow-hidden">
          <header className="sticky top-0 z-10 flex h-14 items-center gap-2 border-b bg-background px-4">
            <SidebarTrigger />
            <div className="ml-auto flex items-center gap-2">
              <Button asChild variant="ghost" size="sm" className="hidden md:inline-flex">
                <Link href="/help" className="inline-flex items-center"><HelpCircle className="mr-2 h-4 w-4" /> Help</Link>
              </Button>
              <DropdownMenu open={feedbackOpen} onOpenChange={setFeedbackOpen}>
                <DropdownMenuTrigger asChild>
                  <Button size="sm" variant="secondary" className="inline-flex items-center">
                    <MessageSquareText className="mr-2 h-4 w-4" /> Feedback
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg p-0" align="end" side="bottom" sideOffset={8}>
                  <div className="p-3">
                    <div className="mb-2 text-sm font-medium">Send feedback</div>
                    <div className="space-y-2">
                      <Label htmlFor="feedback-box" className="text-xs text-muted-foreground">Message</Label>
                      <Textarea id="feedback-box" rows={4} placeholder="Tell us what's on your mind…" value={feedback} onChange={(e) => setFeedback(e.target.value)} />
                    </div>
                    <div className="mt-3 flex items-center justify-end gap-2">
                      <Button variant="ghost" size="sm" onClick={() => setFeedbackOpen(false)}>Cancel</Button>
                      <Button size="sm" onClick={submitFeedback} disabled={sending || !feedback.trim()}>{sending ? "Sending…" : "Send"}</Button>
                    </div>
                  </div>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </header>
          <div className="p-4 md:p-6">{children}</div>
          </div>
        </SidebarInset>
      </SidebarProvider>
    </TeamProvider>
  );
}

function TeamSwitchOverlay() {
  const { switching } = useTeam();
  if (!switching) return null;
  return <LoadingScreen label="Switching team" immediate />;
}
