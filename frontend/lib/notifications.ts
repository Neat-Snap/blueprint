import api from "./api";

export type Notification = {
  id: number;
  createdAt: string;
  updatedAt: string;
  userId: number;
  type: string; // e.g., "workspace_invite"
  data: string; // JSON string
  readAt?: string | null;
};

type AnyRec = Record<string, any>;

function normalize(rec: AnyRec): Notification {
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
  const { data } = await api.get<AnyRec[]>("/notifications");
  return Array.isArray(data) ? data.map(normalize) : [];
}

export async function markNotificationRead(id: number): Promise<{ status: string }> {
  const { data } = await api.patch<{ status: string }>(`/notifications/${id}/read`);
  return data;
}
