import axios from "axios";

// Use a relative base URL so requests go through Next.js rewrites
// This ensures cookies are same-origin in dev and avoids SameSite issues
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || "/api";

const api = axios.create({
  baseURL: `${API_BASE_URL}`,
  withCredentials: true,
});

export default api;
