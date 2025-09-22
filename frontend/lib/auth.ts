import api, { API_BASE_URL } from "./api";

type WorkOSAuthIntent = "login" | "signup";

function redirectToWorkOS(intent: WorkOSAuthIntent) {
  if (typeof window === "undefined") {
    return;
  }
  const base = API_BASE_URL.startsWith("http")
    ? API_BASE_URL
    : `${window.location.origin}${API_BASE_URL.startsWith("/") ? API_BASE_URL : `/${API_BASE_URL}`}`;
  const normalizedBase = base.endsWith("/") ? base : `${base}/`;
  window.location.href = `${normalizedBase}auth/${intent}`;
}

export function beginSignup() {
  redirectToWorkOS("signup");
}

export function beginLogin() {
  redirectToWorkOS("login");
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

export async function resendEmail(email: string): Promise<{ success: boolean; message: string }> {
  const { data } = await api.post<{ success: boolean; message: string }>("/auth/resend-email", { email });
  return data;
}

export async function requestPasswordReset(email: string): Promise<{ success: boolean; message: string }> {
  const { data } = await api.post<{ success: boolean; message: string }>("/auth/password/reset", { email });
  return data;
}

export async function confirmPasswordReset(token: string, password: string): Promise<void> {
  await api.post("/auth/password/confirm", { token, password });
}
