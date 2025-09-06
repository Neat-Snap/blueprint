import axios from "axios";

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || "/api";

const api = axios.create({
  baseURL: `${API_BASE_URL}`,
  withCredentials: true,
});

export default api;

if (typeof window !== "undefined") {
  api.interceptors.response.use(
    (response) => {
      const urlStr = response?.request?.responseURL as string | undefined;
      if (urlStr) {
        try {
          const url = new URL(urlStr, window.location.origin);
          if (url.pathname.startsWith("/auth/verify")) {
            const current = new URL(window.location.href);
            const target = `${url.pathname}${url.search}`;
            const currentPathQuery = `${current.pathname}${current.search}`;
            if (currentPathQuery !== target) {
              window.location.href = target;
            }
            return Promise.reject({ redirectedToVerify: true });
          }
        } catch {
        }
      }

      const ct: string | undefined = response?.headers?.["content-type"] || response?.headers?.["Content-Type"];
      const isHtml = typeof ct === "string" && ct.toLowerCase().includes("text/html");
      if (isHtml) {
        const dataStr = typeof response.data === "string" ? response.data : "";
        if (dataStr.includes("/auth/verify") || dataStr.toLowerCase().includes("<!doctype html")) {
          if (urlStr) {
            try {
              const url = new URL(urlStr, window.location.origin);
              if (url.pathname.startsWith("/auth/verify")) {
                const current = new URL(window.location.href);
                const target = `${url.pathname}${url.search}`;
                const currentPathQuery = `${current.pathname}${current.search}`;
                if (currentPathQuery !== target) {
                  window.location.href = target;
                }
                return Promise.reject({ redirectedToVerify: true });
              }
            } catch {}
          }
          window.location.href = "/auth/verify";
          return Promise.reject({ redirectedToVerify: true });
        }
      }
      return response;
    },
    (error) => {
      const res = error?.response;
      const locationHeader: string | undefined = res?.headers?.location || res?.headers?.Location;
      if ((res?.status === 302 || res?.status === 301) && locationHeader) {
        try {
          const url = new URL(locationHeader, window.location.origin);
          if (url.pathname.startsWith("/auth/verify")) {
            const current = new URL(window.location.href);
            const target = `${url.pathname}${url.search}`;
            const currentPathQuery = `${current.pathname}${current.search}`;
            if (currentPathQuery !== target) {
              window.location.href = target;
            }
            return Promise.reject({ redirectedToVerify: true });
          }
        } catch {
          if (locationHeader.includes("/auth/verify")) {
            try {
              const current = new URL(window.location.href);
              const targetUrl = new URL(locationHeader, window.location.origin);
              const target = `${targetUrl.pathname}${targetUrl.search}`;
              const currentPathQuery = `${current.pathname}${current.search}`;
              if (currentPathQuery !== target) {
                window.location.href = locationHeader;
              }
            } catch {
              if (!window.location.pathname.startsWith("/auth/verify")) {
                window.location.href = locationHeader;
              }
            }
            return Promise.reject({ redirectedToVerify: true });
          }
        }
      }
      return Promise.reject(error);
    }
  );
}
