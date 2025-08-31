import api from "./api";

export type Notification = {
  id: number;
  createdAt: string;
  updatedAt: string;
  userId: number;
  type: string; // e.g., "team_invite"
  data: string; // JSON string
  readAt?: string | null;
};

type BackendRec = {
  id?: number; ID?: number;
  created_at?: string; CreatedAt?: string;
  updated_at?: string; UpdatedAt?: string;
  user_id?: number; UserID?: number;
  type?: string; Type?: string;
  data?: string; Data?: string;
  read_at?: string | null; ReadAt?: string | null;
};

function normalize(rec: BackendRec): Notification {
  return {
    id: rec.id ?? rec.ID,
    createdAt: rec.created_at ?? rec.CreatedAt,
    updatedAt: rec.updated_at ?? rec.UpdatedAt,
    userId: rec.user_id ?? rec.UserID,
    type: rec.type ?? rec.Type,
    data: rec.data ?? rec.Data,
    readAt: rec.read_at ?? rec.ReadAt ?? null,
  } as Notification;
}

export async function listNotifications(): Promise<Notification[]> {
  const { data } = await api.get<BackendRec[]>("/notifications");
  return Array.isArray(data) ? data.map(normalize) : [];
}

export async function markNotificationRead(id: number): Promise<{ status: string }> {
  const { data } = await api.patch<{ status: string }>(`/notifications/${id}/read`);
  return data;
}
