import api from "./api";

export type Overview = {
  user: { id: number; name: string; email: string };
  teams: { id: number; name: string; role: "owner" | "admin" | "regular" }[];
  stats: { total_teams: number; owner_teams: number };
};

export async function getOverview(): Promise<Overview> {
  const { data } = await api.get<Overview>("/dashboard/overview");
  return data;
}
