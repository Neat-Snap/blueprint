"use client";

import React, { useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  getWorkspace,
  updateWorkspaceName,
  deleteWorkspace,
  addMember,
  removeMember,
  reassignOwner,
  type WorkspaceDetail,
} from "@/lib/workspaces";

export default function WorkspaceDetailPage() {
  const router = useRouter();
  const params = useParams<{ id: string }>();
  const id = useMemo(() => Number(params?.id), [params]);

  const [loading, setLoading] = useState(true);
  const [ws, setWs] = useState<WorkspaceDetail | null>(null);

  const [newName, setNewName] = useState("");
  const [savingName, setSavingName] = useState(false);

  const [memberId, setMemberId] = useState("");
  const [memberRole, setMemberRole] = useState<"owner" | "member">("member");
  const [addingMember, setAddingMember] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (!id || Number.isNaN(id)) return;
    (async () => {
      try {
        const data = await getWorkspace(id);
        setWs(data);
        setNewName(data.name);
      } catch {
        router.replace("/auth/login");
        return;
      } finally {
        setLoading(false);
      }
    })();
  }, [id, router]);

  async function onRename(e: React.FormEvent) {
    e.preventDefault();
    if (!ws || !newName.trim()) return;
    setSavingName(true);
    try {
      await updateWorkspaceName(ws.id, newName.trim());
      setWs({ ...ws, name: newName.trim() });
    } finally {
      setSavingName(false);
    }
  }

  async function onDelete() {
    if (!ws) return;
    if (!confirm("Delete this workspace? This cannot be undone.")) return;
    setDeleting(true);
    try {
      await deleteWorkspace(ws.id);
      router.replace("/dashboard/workspaces");
    } finally {
      setDeleting(false);
    }
  }

  async function onAddMember(e: React.FormEvent) {
    e.preventDefault();
    if (!ws) return;
    const uid = Number(memberId);
    if (!uid || Number.isNaN(uid)) return;
    setAddingMember(true);
    try {
      await addMember(ws.id, uid, memberRole);
      // optimistic add
      setWs({
        ...ws,
        members: [
          ...ws.members,
          { id: uid as any, name: String(uid), role: memberRole },
        ],
      });
      setMemberId("");
      setMemberRole("member");
    } finally {
      setAddingMember(false);
    }
  }

  async function onRemoveMember(uid: number) {
    if (!ws) return;
    await removeMember(ws.id, uid);
    setWs({ ...ws, members: ws.members.filter((m) => m.id !== uid) });
  }

  async function onReassignOwner(uid: number) {
    if (!ws) return;
    await reassignOwner(ws.id, uid);
    // After reassignment, navigate back or refresh
    const fresh = await getWorkspace(ws.id);
    setWs(fresh);
  }

  if (loading || !ws) return null;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Workspace</h1>
        <p className="text-muted-foreground">Manage workspace settings and members.</p>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Members</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {ws.members.length ? (
              <ul className="divide-y">
                {ws.members.map((m) => (
                  <li key={m.id} className="flex items-center justify-between py-3">
                    <div>
                      <div className="font-medium">{m.name}</div>
                      <div className="text-xs text-muted-foreground">Role: {m.role}</div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button variant="outline" size="sm" onClick={() => onRemoveMember(m.id)}>
                        Remove
                      </Button>
                      <Button variant="secondary" size="sm" onClick={() => onReassignOwner(m.id)}>
                        Make owner
                      </Button>
                    </div>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground">No members yet.</p>
            )}

            <form onSubmit={onAddMember} className="grid gap-2 sm:grid-cols-3 sm:items-end">
              <div className="space-y-1">
                <Label htmlFor="memberId">User ID</Label>
                <Input id="memberId" placeholder="e.g. 42" value={memberId} onChange={(e) => setMemberId(e.target.value)} />
              </div>
              <div className="space-y-1">
                <Label>Role</Label>
                <Select value={memberRole} onValueChange={(v: any) => setMemberRole(v)}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select role" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="member">Member</SelectItem>
                    <SelectItem value="owner">Owner</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Button type="submit" className="w-full" disabled={addingMember || !memberId.trim()}>
                  {addingMember ? "Adding..." : "Add member"}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Settings</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <form onSubmit={onRename} className="space-y-2">
              <Label htmlFor="name">Workspace name</Label>
              <Input id="name" value={newName} onChange={(e) => setNewName(e.target.value)} />
              <Button type="submit" disabled={savingName || !newName.trim()} className="w-full">
                {savingName ? "Saving..." : "Save name"}
              </Button>
            </form>
            <hr className="my-2" />
            <Button variant="destructive" onClick={onDelete} disabled={deleting} className="w-full">
              {deleting ? "Deleting..." : "Delete workspace"}
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
