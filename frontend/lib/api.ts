import axios from "axios";

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

const api = axios.create({
  baseURL: `${API_BASE_URL}`,
  withCredentials: true,
});

export default api;
