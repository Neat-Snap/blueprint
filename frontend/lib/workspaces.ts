import api from "./api";

export type Workspace = { id: number; name: string; icon?: string; owner_id: number };
export type WorkspaceDetail = {
  id: number;
  name: string;
  icon?: string;
  owner_id: number;
  members: { id: number; name: string; role: string }[];
};

export async function listWorkspaces(): Promise<Workspace[]> {
  const { data } = await api.get<Workspace[]>("/workspaces");
  return data;
}

export async function createWorkspace(name: string, icon?: string): Promise<{ id: number; name: string; icon?: string; role: string }> {
  const { data } = await api.post<{ id: number; name: string; icon?: string; role: string }>("/workspaces", { name, icon });
  return data;
}

export async function getWorkspace(id: number): Promise<WorkspaceDetail> {
  const { data } = await api.get<WorkspaceDetail>(`/workspaces/${id}`);
  return data;
}

export async function updateWorkspaceName(id: number, name: string, icon?: string): Promise<{ success: boolean; status: string }> {
  const { data } = await api.patch<{ success: boolean; status: string }>(`/workspaces/${id}`, { name, icon });
  return data;
}

export async function deleteWorkspace(id: number): Promise<{ success: boolean; status: string }> {
  const { data } = await api.delete<{ success: boolean; status: string }>(`/workspaces/${id}`);
  return data;
}

export async function addMember(workspaceId: number, userId: number, role: "owner" | "member"): Promise<{ success: boolean; status: string }> {
  const { data } = await api.post<{ success: boolean; status: string }>(`/workspaces/${workspaceId}/members`, {
    user_id: userId,
    role,
  });
  return data;
}

export async function removeMember(workspaceId: number, userId: number): Promise<{ success: boolean; status: string }> {
  const { data } = await api.delete<{ success: boolean; status: string }>(`/workspaces/${workspaceId}/members/${userId}`);
  return data;
}

export async function reassignOwner(workspaceId: number, userId: number): Promise<{ success: boolean; status: string }> {
  const { data } = await api.post<{ success: boolean; status: string }>(`/workspaces/${workspaceId}/owner`, {
    user_id: userId,
  });
  return data;
}

export type WorkspaceOverview = {
  workspace: { id: number; name: string; icon?: string };
  stats: { members_count: number };
};

export async function getWorkspaceOverview(id: number): Promise<WorkspaceOverview> {
  const { data } = await api.get<WorkspaceOverview>(`/workspaces/${id}/overview`);
  return data;
}
