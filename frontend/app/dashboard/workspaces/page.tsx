"use client";

import React, { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { listWorkspaces, createWorkspace, type Workspace } from "@/lib/workspaces";

export default function WorkspacesIndexPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<Workspace[]>([]);
  const [name, setName] = useState("");
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const data = await listWorkspaces();
        setItems(data);
      } catch {
        router.replace("/auth/login");
        return;
      } finally {
        setLoading(false);
      }
    })();
  }, [router]);

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setCreating(true);
    try {
      const created = await createWorkspace(name.trim());
      setItems((prev) => [...prev, { id: created.id, name: created.name, owner_id: 0 }]);
      setName("");
    } finally {
      setCreating(false);
    }
  }

  if (loading) return null;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Workspaces</h1>
        <p className="text-muted-foreground">Manage all your workspaces.</p>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Your workspaces</CardTitle>
          </CardHeader>
          <CardContent>
            {items.length ? (
              <ul className="divide-y">
                {items.map((w) => (
                  <li key={w.id} className="flex items-center justify-between py-3">
                    <div>
                      <div className="font-medium">{w.name}</div>
                      <div className="text-xs text-muted-foreground">ID: {w.id}</div>
                    </div>
                    <Button variant="outline" size="sm" onClick={() => router.push(`/dashboard/workspaces/${w.id}`)}>
                      Open
                    </Button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground">No workspaces yet.</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Create workspace</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={onCreate} className="space-y-3">
              <Input placeholder="Workspace name" value={name} onChange={(e) => setName(e.target.value)} />
              <Button type="submit" disabled={creating || !name.trim()} className="w-full">
                {creating ? "Creating..." : "Create"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
