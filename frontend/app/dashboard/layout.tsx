"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { LayoutDashboard, Users2, Settings2, Plus } from "lucide-react";

import { AppSidebar, type NavMainItem, type ProjectItem, type SecondaryItem } from "@/components/app-sidebar";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { getMe } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { WorkspaceProvider } from "@/lib/workspace-context";
import { WorkspaceSwitcher } from "@/components/workspace-switcher";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [user, setUser] = useState<{ name: string; email: string; avatar: string }>({ name: "", email: "", avatar: "" });

  useEffect(() => {
    (async () => {
      try {
        const me = await getMe();
        setUser({ name: me.name || "", email: me.email || "", avatar: "" });
      } catch {
        // noop; page's guards will redirect if needed
      }
    })();
  }, []);

  const navMain: NavMainItem[] = [
    { title: "Dashboard", url: "/dashboard", icon: LayoutDashboard, isActive: pathname === "/dashboard" },
    { title: "Settings", url: "/dashboard/settings", icon: Settings2, isActive: pathname?.startsWith("/dashboard/settings") },
  ];

  const projects: ProjectItem[] = [];
  const navSecondary: SecondaryItem[] = [];

  return (
    <WorkspaceProvider>
      <SidebarProvider>
        <AppSidebar
          org={{ name: "Your App", plan: "Free", href: "/dashboard" }}
          user={{ name: user.name || user.email || "User", email: user.email || "", avatar: user.avatar }}
          navMain={navMain}
          projects={projects}
          navSecondary={navSecondary}
          headerSlot={<WorkspaceSwitcher />}
        />
        <SidebarInset>
          <header className="sticky top-0 z-10 flex h-14 items-center gap-2 border-b bg-background px-4">
            <SidebarTrigger />
            <div className="ml-auto flex items-center gap-2">
              <Button asChild size="sm" className="hidden md:inline-flex">
                <Link href="/dashboard/workspaces/new"><Plus className="mr-2 h-4 w-4" /> New workspace</Link>
              </Button>
            </div>
          </header>
          <div className="p-4 md:p-6">{children}</div>
        </SidebarInset>
      </SidebarProvider>
    </WorkspaceProvider>
  );
}
