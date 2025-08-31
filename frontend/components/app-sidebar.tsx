"use client"

import * as React from "react"
import { Command, type LucideIcon } from "lucide-react"
import Link from "next/link"

import { NavMain } from "@/components/nav-main"
import { NavProjects } from "@/components/nav-projects"
import { NavSecondary } from "@/components/nav-secondary"
import { NavUser } from "@/components/nav-user"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

export type NavMainItem = {
  title: string
  url: string
  icon: LucideIcon
  isActive?: boolean
  items?: { title: string; url: string }[]
}

export type ProjectItem = {
  name: string
  url: string
  icon: LucideIcon
}

export type SecondaryItem = {
  title: string
  url: string
  icon: LucideIcon
}

export type AppSidebarProps = React.ComponentProps<typeof Sidebar> & {
  org?: { name: string; plan?: string; href?: string }
  user: { name: string; email: string; avatar: string }
  navMain: NavMainItem[]
  projects?: ProjectItem[]
  navSecondary?: SecondaryItem[]
  headerSlot?: React.ReactNode
}

export function AppSidebar({ org, user, navMain, projects = [], navSecondary = [], headerSlot, ...props }: AppSidebarProps) {
  return (
    <Sidebar variant="inset" collapsible="icon" {...props}>
      <SidebarHeader>
        {headerSlot ? (
          headerSlot
        ) : (
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton size="lg" asChild>
                <Link href={org?.href || "/dashboard"}>
                  <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg">
                    <Command className="size-4" />
                  </div>
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">{org?.name || "Team"}</span>
                    <span className="truncate text-xs">{org?.plan || "Free"}</span>
                  </div>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        )}
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={navMain} />
        {projects.length > 0 && <NavProjects projects={projects} />}
        {navSecondary.length > 0 && <NavSecondary items={navSecondary} className="mt-auto" />}
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={user} />
      </SidebarFooter>
    </Sidebar>
  )
}
