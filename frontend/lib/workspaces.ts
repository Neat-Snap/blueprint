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

export async function addMember(workspaceId: number, userId: number, role: "regular" | "admin"): Promise<{ success: boolean; status: string }> {
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

export async function updateMemberRole(
  workspaceId: number,
  userId: number,
  role: "regular" | "admin"
): Promise<{ status: string }> {
  const { data } = await api.patch<{ status: string }>(
    `/workspaces/${workspaceId}/members/${userId}/role`,
    { role }
  );
  return data;
}

// owner reassignment removed

export type WorkspaceOverview = {
  workspace: { id: number; name: string; icon?: string };
  stats: { members_count: number };
};

export async function getWorkspaceOverview(id: number): Promise<WorkspaceOverview> {
  const { data } = await api.get<WorkspaceOverview>(`/workspaces/${id}/overview`);
  return data;
}

// Invitations
export async function createInvitation(workspaceId: number, email: string, role: "regular" | "admin" = "regular"): Promise<{ token: string }> {
  const { data } = await api.post<{ token: string }>(`/workspaces/${workspaceId}/invitations`, { email, role });
  return data;
}

export async function acceptInvitation(token: string): Promise<{ status: string }> {
  const { data } = await api.post<{ status: string }>(`/workspaces/invitations/accept`, { token });
  return data;
}

export type WorkspaceInvitation = {
  id: number;
  email: string;
  role: "regular" | "admin" | string;
  token: string;
  status: string;
  created_at: string;
  expires_at: string;
};

export async function listInvitations(workspaceId: number): Promise<WorkspaceInvitation[]> {
  const { data } = await api.get<WorkspaceInvitation[]>(`/workspaces/${workspaceId}/invitations`);
  return data;
}

export async function revokeInvitation(workspaceId: number, invitationId: number): Promise<{ status: string }> {
  const { data } = await api.delete<{ status: string }>(`/workspaces/${workspaceId}/invitations/${invitationId}`);
  return data;
}
