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
  // Redirect browser to backend OAuth begin endpoint
  window.location.href = `${API_BASE_URL}/auth/google`;
}

export async function getMe() {
  const { data } = await api.get<{ id?: string; email?: string; name?: string }>("/auth/me");
  return data;
}

export async function logout() {
  // Backend may have GET /auth/logout that clears cookie. If not, this will 404 and we just navigate client-side.
  try {
    await api.get("/auth/logout");
  } catch (_) {
    // ignore
  }
}
