import api from "./api";

export type UpdateProfilePayload = {
  name: string;
  avatar_url: string;
};

export async function updateProfile(payload: UpdateProfilePayload) {
  const { data } = await api.patch<UpdateProfilePayload>("/account/profile", payload);
  return data;
}

export async function changePassword(current_password: string, new_password: string) {
  await api.post("/account/password/change", { current_password, new_password });
}

export async function changeEmail(email: string): Promise<{ confirmation_id: string }> {
  const { data } = await api.post<{ confirmation_id: string }>("/accounts/email/change", { email });
  return data;
}
