"use client";

import React, { useEffect, useState } from "react";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useWorkspace } from "@/lib/workspace-context";
import { getWorkspace, updateWorkspaceName, addMember, removeMember, reassignOwner, deleteWorkspace } from "@/lib/workspaces";
import { Trash2, Crown } from "lucide-react";

export default function WorkspaceSettingsPanel() {
  const { current, refresh, setCurrentId, all } = useWorkspace();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [name, setName] = useState("");
  const [ownerId, setOwnerId] = useState<number | null>(null);
  const [members, setMembers] = useState<{ id: number; name: string; role: string }[]>([]);
  const [newMemberId, setNewMemberId] = useState("");
  const [newMemberRole, setNewMemberRole] = useState<"member" | "owner">("member");

  useEffect(() => {
    (async () => {
      if (!current) {
        setLoading(false);
        return;
      }
      try {
        const data = await getWorkspace(current.id);
        setName(data.name);
        setOwnerId(data.owner_id);
        setMembers(data.members);
      } finally {
        setLoading(false);
      }
    })();
  }, [current]);

  async function handleRename() {
    if (!current || !name.trim()) return;
    setSaving(true);
    try {
      await updateWorkspaceName(current.id, name.trim());
      await refresh();
    } finally {
      setSaving(false);
    }
  }

  async function handleAddMember() {
    if (!current) return;
    const idNum = Number(newMemberId);
    if (!idNum) return;
    await addMember(current.id, idNum, newMemberRole);
    const data = await getWorkspace(current.id);
    setMembers(data.members);
    setOwnerId(data.owner_id);
    setNewMemberId("");
    setNewMemberRole("member");
  }

  async function handleRemoveMember(uid: number) {
    if (!current) return;
    await removeMember(current.id, uid);
    setMembers((m) => m.filter((x) => x.id !== uid));
  }

  async function handleMakeOwner(uid: number) {
    if (!current) return;
    await reassignOwner(current.id, uid);
    const data = await getWorkspace(current.id);
    setMembers(data.members);
    setOwnerId(data.owner_id);
  }

  async function handleDelete() {
    if (!current) return;
    await deleteWorkspace(current.id);
    // pick another workspace if available
    await refresh();
    const remaining = all.filter((w) => w.id !== current.id);
    setCurrentId(remaining[0]?.id ?? null);
  }

  if (loading) return null;
  if (!current) return <p className="text-sm text-muted-foreground">Select a workspace from the header to manage settings.</p>;

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>General</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="ws-name">Name</Label>
            <Input id="ws-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Acme Corp" />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button onClick={handleRename} disabled={!name.trim() || saving}>{saving ? "Saving..." : "Save changes"}</Button>
        </CardFooter>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Members</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-2">
            {members.length ? (
              members.map((m) => (
                <div key={m.id} className="flex items-center justify-between rounded border p-2 text-sm">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{m.name || `User #${m.id}`}</span>
                    {ownerId === m.id ? <span className="text-xs text-muted-foreground">(owner)</span> : null}
                  </div>
                  <div className="flex items-center gap-2">
                    {ownerId !== m.id && (
                      <Button size="sm" variant="secondary" onClick={() => handleMakeOwner(m.id)}>
                        <Crown className="mr-1 h-4 w-4" /> Make owner
                      </Button>
                    )}
                    <Button size="sm" variant="destructive" onClick={() => handleRemoveMember(m.id)}>
                      <Trash2 className="mr-1 h-4 w-4" /> Remove
                    </Button>
                  </div>
                </div>
              ))
            ) : (
              <p className="text-sm text-muted-foreground">No members.</p>
            )}
          </div>

          <div className="mt-2 flex items-end gap-2">
            <div className="flex-1">
              <Label htmlFor="new-member">Add member (User ID)</Label>
              <Input id="new-member" value={newMemberId} onChange={(e) => setNewMemberId(e.target.value)} placeholder="123" />
            </div>
            <div>
              <Label className="sr-only">Role</Label>
              <Select value={newMemberRole} onValueChange={(v) => setNewMemberRole(v as any)}>
                <SelectTrigger className="w-[140px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="member">Member</SelectItem>
                  <SelectItem value="owner">Owner</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <Button onClick={handleAddMember}>Add</Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Danger zone</CardTitle>
        </CardHeader>
        <CardContent className="flex items-center justify-between">
          <div>
            <p className="text-sm">Delete this workspace</p>
            <p className="text-xs text-muted-foreground">This action is irreversible.</p>
          </div>
          <Button variant="destructive" onClick={handleDelete}><Trash2 className="mr-1 h-4 w-4" /> Delete</Button>
        </CardContent>
      </Card>
    </div>
  );
}
