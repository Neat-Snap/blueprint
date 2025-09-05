import axios from "axios";

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || "/api";

const api = axios.create({
  baseURL: `${API_BASE_URL}`,
  withCredentials: true,
});

export default api;
