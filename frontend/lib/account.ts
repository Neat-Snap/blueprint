import api from "./api";

export interface ProfileUpdateRequest {
  name: string;
  avatar_url: string;
}

export interface ProfileUpdateResponse {
  name: string;
  avatar_url: string;
}

export async function updateProfile(body: ProfileUpdateRequest) {
  const { data } = await api.patch<ProfileUpdateResponse>("/account/profile", body);
  return data;
}

export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
}

export async function changePassword(body: ChangePasswordRequest) {
  // backend returns success envelope; we don't depend on its shape
  await api.patch("/account/password/change", body);
}

export interface ChangeEmailRequest {
  email: string;
}
export interface ChangeEmailResponse {
  confirmation_id: string;
}

export async function changeEmail(body: ChangeEmailRequest) {
  const { data } = await api.patch<ChangeEmailResponse>("/account/email/change", body);
  return data;
}

export interface ConfirmEmailRequest {
  confirmation_id: string;
  code: string;
}

export async function confirmEmail(body: ConfirmEmailRequest) {
  await api.patch("/account/email/confirm", body);
}
