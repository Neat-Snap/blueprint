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

export type UserPreferences = {
  theme?: "light" | "dark" | "system";
  language?: string;
};

export async function getPreferences() {
  const { data } = await api.get<{ theme?: "light" | "dark" | "system"; lang?: string }>("/account/preferences");
  // Map backend 'lang' to frontend 'language'
  return { theme: data.theme, language: data.lang } satisfies UserPreferences;
}

export async function updateTheme(theme: "light" | "dark" | "system") {
  await api.post("/account/preferences/theme", { theme });
}

export async function updateLanguage(language: string) {
  await api.post("/account/preferences/language", { lang: language });
}
