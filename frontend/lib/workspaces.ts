import api from "./api";

export interface WorkspaceSummary {
  id: number;
  name: string;
  owner_id: number;
}

export interface WorkspaceCreated {
  id: number;
  name: string;
  role: "owner" | "member";
}

export interface WorkspaceMember {
  id: number;
  name: string;
  role: "owner" | "member";
}

export interface WorkspaceDetail {
  id: number;
  name: string;
  owner_id: number;
  members: WorkspaceMember[];
}

export async function listWorkspaces() {
  const { data } = await api.get<WorkspaceSummary[]>("/workspaces/");
  return data;
}

export async function createWorkspace(name: string) {
  const { data } = await api.post<WorkspaceCreated>("/workspaces/", { name });
  return data;
}

export async function getWorkspace(id: number) {
  const { data } = await api.get<WorkspaceDetail>(`/workspaces/${id}`);
  return data;
}

export async function updateWorkspaceName(id: number, name: string) {
  const { data } = await api.patch<{ success: boolean; status: string }>(`/workspaces/${id}`, { name });
  return data;
}

export async function deleteWorkspace(id: number) {
  const { data } = await api.delete<{ success: boolean; status: string }>(`/workspaces/${id}`);
  return data;
}

export async function addMember(workspaceId: number, userId: number, role: "owner" | "member") {
  const { data } = await api.post<{ success: boolean; status: string }>(`/workspaces/${workspaceId}/members`, { user_id: userId, role });
  return data;
}

export async function removeMember(workspaceId: number, userId: number) {
  const { data } = await api.delete<{ success: boolean; status: string }>(`/workspaces/${workspaceId}/members/${userId}`);
  return data;
}

export async function reassignOwner(workspaceId: number, userId: number) {
  const { data } = await api.post<{ success: boolean; status: string }>(`/workspaces/${workspaceId}/owner`, { user_id: userId });
  return data;
}
