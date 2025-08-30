import api from "./api";

export async function updateProfile(name: string, avatar_url: string) {
  const { data } = await api.patch<{ name: string; avatar_url: string }>("/account/profile", {
    name,
    avatar_url,
  });
  return data;
}

export async function changeEmail(email: string) {
  const { data } = await api.patch<{ confirmation_id: string }>("/account/email/change", { email });
  return data; // { confirmation_id }
}

export async function confirmEmail(confirmation_id: string, code: string) {
  await api.patch("/account/email/confirm", { confirmation_id, code });
}

export async function changePassword(current_password: string, new_password: string) {
  await api.patch("/account/password/change", { current_password, new_password });
}
