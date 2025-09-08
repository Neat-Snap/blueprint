import api from "./api";

export type Team = { id: number; name: string; icon?: string; owner_id: number };
export type TeamDetail = {
  id: number;
  name: string;
  icon?: string;
  owner_id: number;
  members: { id: number; name: string; email: string; role: string }[];
};

export async function listTeams(): Promise<Team[]> {
  const { data } = await api.get<Team[]>("/teams");
  return data;
}

export async function createTeam(name: string, icon?: string): Promise<{ id: number; name: string; icon?: string; role: string }> {
  const { data } = await api.post<{ id: number; name: string; icon?: string; role: string }>("/teams", { name, icon });
  return data;
}

export async function getTeam(id: number): Promise<TeamDetail> {
  const { data } = await api.get<TeamDetail>(`/teams/${id}`);
  return data;
}

export async function updateTeam(id: number, name: string, icon?: string): Promise<{ success: boolean; status: string }> {
  const { data } = await api.patch<{ success: boolean; status: string }>(`/teams/${id}`, { name, icon });
  return data;
}

export async function deleteTeam(id: number): Promise<{ success: boolean; status: string }> {
  const { data } = await api.delete<{ success: boolean; status: string }>(`/teams/${id}`);
  return data;
}

export async function addMember(teamId: number, userId: number, role: "regular" | "admin"): Promise<{ success: boolean; status: string }> {
  const { data } = await api.post<{ success: boolean; status: string }>(`/teams/${teamId}/members`, {
    user_id: userId,
    role,
  });
  return data;
}

export async function removeMember(teamId: number, userId: number): Promise<{ success: boolean; status: string }> {
  const { data } = await api.delete<{ success: boolean; status: string }>(`/teams/${teamId}/members/${userId}`);
  return data;
}

export async function updateMemberRole(
  teamId: number,
  userId: number,
  role: "regular" | "admin"
): Promise<{ status: string }> {
  const { data } = await api.patch<{ status: string }>(
    `/teams/${teamId}/members/${userId}/role`,
    { role }
  );
  return data;
}

// owner reassignment removed

export type TeamOverview = {
  team: { id: number; name: string; icon?: string };
  stats: { members_count: number };
};

export async function getTeamOverview(id: number): Promise<TeamOverview> {
  const { data } = await api.get<TeamOverview>(`/teams/${id}/overview`);
  return data;
}

// Invitations
export async function createInvitation(teamId: number, email: string, role: "regular" | "admin" = "regular"): Promise<{ token: string }> {
  const { data } = await api.post<{ token: string }>(`/teams/${teamId}/invitations`, { email, role });
  return data;
}

export async function acceptInvitation(token: string): Promise<{ status: string; team_id: number; team_name: string; role: string }> {
  const { data } = await api.post<{ status: string; team_id: number; team_name: string; role: string }>(`/teams/invitations/accept`, { token });
  return data;
}

export type InvitationStatus = {
  status: string; // pending | revoked | accepted | expired
  team_id: number;
  team_name: string;
  role: string;
  expires_at: string;
};

export async function checkInvitationStatus(token: string): Promise<InvitationStatus> {
  const { data } = await api.post<InvitationStatus>(`/teams/invitations/check`, { token });
  return data;
}

export type TeamInvitation = {
  id: number;
  email: string;
  role: "regular" | "admin" | string;
  token?: string;
  status: string;
  created_at: string;
  expires_at: string;
};

export async function listInvitations(teamId: number): Promise<TeamInvitation[]> {
  const { data } = await api.get<TeamInvitation[]>(`/teams/${teamId}/invitations`);
  return data;
}

export async function revokeInvitation(teamId: number, invitationId: number): Promise<{ status: string }> {
  const { data } = await api.delete<{ status: string }>(`/teams/${teamId}/invitations/${invitationId}`);
  return data;
}
