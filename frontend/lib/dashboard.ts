import api from "./api";

export interface DashboardOverviewResponse {
  user: { id: number; name: string; email: string };
  workspaces: Array<{ id: number; name: string; role: "owner" | "member" }>;
  stats: { total_workspaces: number; owner_workspaces: number };
}

export async function getOverview() {
  const { data } = await api.get<DashboardOverviewResponse>("/dashboard/overview");
  return data;
}
