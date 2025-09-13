import api, { API_BASE_URL } from "./api";
import type { SignupResponse } from "./types";

export async function signup(email: string, password: string) {
  const { data } = await api.post<SignupResponse>("/auth/signup", { email, password });
  return data;
}

export async function confirmEmail(confirmation_id: string, code: string): Promise<void> {
  await api.post("/auth/confirm-email", { confirmation_id, code });
}

export async function login(email: string, password: string): Promise<void> {
  await api.post("/auth/login", { email, password });
}

export function beginGoogleLogin() {
  window.location.href = `${API_BASE_URL}/auth/google`;
}

export function beginGithubLogin() {
  window.location.href = `${API_BASE_URL}/auth/github`;
}

export async function getMe() {
  const { data } = await api.get<{ id?: string; email?: string; name?: string }>("/auth/me");
  return data;
}

export async function logout() {
  try {
    await api.get("/auth/logout");
  } catch {
  }
}

export async function resendEmail(email: string): Promise<SignupResponse> {
  const { data } = await api.post<SignupResponse>("/auth/resend-email", { email });
  return data;
}

export async function requestPasswordReset(email: string): Promise<{ success: boolean; message: string }>{
  const { data } = await api.post<{ success: boolean; message: string }>("/auth/password/reset", { email });
  return data;
}

export async function confirmPasswordReset(reset_password_id: string, code: string, password: string): Promise<void> {
  // Backend may respond with a redirect (302). Axios treats it as success; we don't need the response body.
  await api.post("/auth/password/confirm", { reset_password_id, code, password });
}
