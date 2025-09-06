"use client";

import React, { useEffect, useRef, useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { acceptInvitation } from "@/lib/teams";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { useTeam } from "@/lib/teams-context";
import { toast } from "sonner";

export default function InviteAcceptPage() {
  const search = useSearchParams();
  const router = useRouter();
  const token = search.get("token") || "";
  const [status, setStatus] = useState<"idle" | "pending" | "success" | "error">("idle");
  const [error, setError] = useState<string | null>(null);
  const ranOnceRef = useRef<string | null>(null);
  const { refresh, switchTo } = useTeam();

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!token) return;

      if (ranOnceRef.current === token) return;
      ranOnceRef.current = token;
      setStatus("pending");
      setError(null);
      try {
        const res = await acceptInvitation(token);
        if (cancelled) return;
        setStatus("success");

        // Ensure the new team appears in the list and becomes active
        await refresh();
        await switchTo(res.team_id);
        toast.success(`Invitation accepted`, { description: `You're now a member of ${res.team_name}.` });
        setTimeout(() => {
          router.push("/dashboard");
        }, 300);
      } catch (e: unknown) {
        const err = e as { response?: { data?: { error?: string } }; message?: string };
        if (cancelled) return;
        setStatus("error");
        setError(err.response?.data?.error || err.message || "Failed to accept invitation");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token, router]);

  const retry = async () => {
    if (!token) return;
    setStatus("pending");
    setError(null);
    try {
      const res = await acceptInvitation(token);
      setStatus("success");
      await refresh();
      await switchTo(res.team_id);
      toast.success(`Invitation accepted`, { description: `You're now a member of ${res.team_name}.` });
      setTimeout(() => router.push("/dashboard"), 300);
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } }; message?: string };
      setStatus("error");
      setError(err.response?.data?.error || err.message || "Failed to accept invitation");
    }
  };

  return (
    <div className="mx-auto mt-16 max-w-md px-4">
      <Card>
        <CardHeader>
          <CardTitle>Accept Invitation</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {!token && (
            <p className="text-sm text-muted-foreground">No token found in URL. Please use the link provided in your invitation email.</p>
          )}

          {token && status === "idle" && (
            <p className="text-sm text-muted-foreground">Preparing to accept your invitation…</p>
          )}

          {status === "pending" && (
            <p className="text-sm">Accepting invitation…</p>
          )}

          {status === "success" && (
            <p className="text-sm">Invitation accepted! Redirecting you to your dashboard…</p>
          )}

          {status === "error" && (
            <div className="space-y-2">
              <p className="text-sm text-red-600">{error}</p>
              <div className="flex gap-2">
                <Button variant="secondary" onClick={retry}>Try again</Button>
                <Button variant="outline" onClick={() => router.push("/")}>Go home</Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
