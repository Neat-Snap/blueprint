import api from "./api";

export type Overview = {
  user: { id: number; name: string; email: string };
  workspaces: { id: number; name: string; role: "owner" | "member" }[];
  stats: { total_workspaces: number; owner_workspaces: number };
};

export async function getOverview(): Promise<Overview> {
  const { data } = await api.get<Overview>("/dashboard/overview");
  return data;
}
