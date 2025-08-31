"use client";

import React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";

export default function HelpPage() {
  return (
    <div className="mx-auto max-w-5xl space-y-8 p-6 md:p-8">
      <div className="space-y-1">
        <h1 className="text-3xl font-semibold tracking-tight">Help Center</h1>
        <p className="text-muted-foreground">Answers to common questions, setup guides, and ways to get in touch.</p>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Frequently Asked Questions</CardTitle>
          </CardHeader>
          <CardContent>
            <Accordion type="single" collapsible className="w-full">
              <AccordionItem value="item-1">
                <AccordionTrigger>How do I create or join a team?</AccordionTrigger>
                <AccordionContent>
                  Owners can create teams from the Team switcher. To join, accept an email invitation from an owner or admin. Invitations also appear in your Notifications.
                </AccordionContent>
              </AccordionItem>
              <AccordionItem value="item-2">
                <AccordionTrigger>What are the team roles?</AccordionTrigger>
                <AccordionContent>
                  Roles are owner, admin, and regular. Owners can manage billing, deletion, and member roles. Admins manage settings and members (except roles). Regular users have standard access.
                </AccordionContent>
              </AccordionItem>
              <AccordionItem value="item-3">
                <AccordionTrigger>How can I change a member&#39;s role?</AccordionTrigger>
                <AccordionContent>
                  Only the team owner can change roles in <span className="font-medium">Settings → Members</span> using the role dropdown next to each member.
                </AccordionContent>
              </AccordionItem>
              <AccordionItem value="item-4">
                <AccordionTrigger>I didn’t receive an email</AccordionTrigger>
                <AccordionContent>
                  Check spam, then verify your email is correct in Account settings. You can re-send confirmation emails. If issues persist, contact support.
                </AccordionContent>
              </AccordionItem>
              <AccordionItem value="item-5">
                <AccordionTrigger>How do I report a bug or request a feature?</AccordionTrigger>
                <AccordionContent>
                  Use the <span className="font-medium">Feedback</span> button in the top bar. Provide steps to reproduce or your feature idea for quickest turnaround.
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Quick Start</CardTitle>
          </CardHeader>
          <CardContent className="text-sm">
            <ol className="list-decimal space-y-2 pl-5 text-muted-foreground">
              <li>Create or pick a team from the switcher in the sidebar.</li>
              <li>Invite teammates from <span className="font-medium text-foreground">Settings → Members</span>.</li>
              <li>Assign roles and configure team settings.</li>
              <li>Explore the Dashboard to track activity and notifications.</li>
            </ol>
            <Separator className="my-4" />
            <div className="text-foreground">Tip: Use ⌘/Ctrl + 1..9 to quickly switch teams.</div>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Account & Billing</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 text-sm text-muted-foreground">
            <div>
              <div className="font-medium text-foreground">Change email or password</div>
              Update these in Account settings. You’ll need to confirm email changes via a verification link.
            </div>
            <Separator />
            <div>
              <div className="font-medium text-foreground">Managing members</div>
              Owners and admins can invite/remove members. Only owners can change roles and delete the team.
            </div>
            <Separator />
            <div>
              <div className="font-medium text-foreground">Exporting data</div>
              Contact support via Feedback to request data exports while we finalize in-app export tools.
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Resources</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            <ul className="list-disc space-y-2 pl-5">
              <li>Product updates: see Dashboard notifications</li>
              <li>Status: surfaced in-app if there are issues</li>
              <li>Security: we use email confirmation and session protection</li>
            </ul>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Contact</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <p className="text-muted-foreground">Can’t find what you need? Use the Feedback button in the top bar to reach us.</p>
          <ul className="list-disc pl-6 text-muted-foreground">
            <li>Support: configured via <span className="font-mono">SUPPORT_EMAIL</span></li>
            <li>Developers: optionally via <span className="font-mono">DEVELOPER_EMAIL</span></li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
