import axios, { AxiosError, InternalAxiosRequestConfig } from "axios";

export const API_BASE = import.meta.env.VITE_API_BASE_URL || "/api/v1";
export const API_ROOT = API_BASE.replace(/\/api\/v1\/?$/, "");

export function getMediaUrl(path: string, bookId?: string, updatedAt?: string | number): string {
  if (!path) return "";
  if (path.startsWith("blob:") || path.startsWith("data:")) return path;
  if (path.startsWith("http")) {
    const suffix = bookId ? `&book_id=${bookId}` : "";
    return `${API_BASE}/reader/proxy-cover?url=${encodeURIComponent(path)}${suffix}`;
  }
  const cleanPath = path.startsWith("/") ? path : `/${path}`;
  const finalPath = cleanPath.replace(/^\/data\//, '/');
  let url = `${API_ROOT}${finalPath}`;
  if (updatedAt) {
    const timeVal = typeof updatedAt === "number" ? updatedAt : new Date(updatedAt).getTime();
    if (!isNaN(timeVal) && timeVal > 0) {
      const sep = url.includes("?") ? "&" : "?";
      url = `${url}${sep}t=${timeVal}`;
    }
  }
  return url;
}

export function toQuery(params: Record<string, unknown>) {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") continue;
    if (Array.isArray(value)) {
      for (const item of value) search.append(key, String(item));
    } else {
      search.set(key, String(value));
    }
  }
  const query = search.toString();
  return query ? `?${query}` : "";
}

export const api = axios.create({
  baseURL: API_BASE,
  withCredentials: true,
});

interface CustomAxiosRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean;
}

interface QueueItem {
  resolve: () => void;
  reject: (error: unknown) => void;
}

let isRefreshing = false;
let refreshFailedAt = 0;
let queue: QueueItem[] = [];

function getCookie(name: string): string | null {
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(";").shift() || null;
  return null;
}

api.interceptors.request.use((config) => {
  const csrfToken = getCookie("csrf_token");
  if (csrfToken && config.headers) {
    config.headers["X-CSRF-Token"] = csrfToken;
  }
  return config;
});

const processQueue = (error: unknown = null) => {
  queue.forEach((p) => {
    if (error) p.reject(error);
    else p.resolve();
  });
  queue = [];
};

const skipRefreshUrls = [
  "/auth/signin",
  "/auth/signup",
  "/auth/register",
  "/auth/logout",
  "/auth/refresh",
  "/auth/forgot-password",
  "/setup",
];

const authActionPatterns = [
  "/auth/signin",
  "/auth/signup",
  "/auth/register",
  "/auth/magic-code",
  "/auth/oauth2",
  "/auth/otp/verify",
];

const REFRESH_COOLDOWN_MS = 30_000;

api.interceptors.response.use(
  (res) => {
    const url = res.config.url || "";
    if (authActionPatterns.some((p) => url.includes(p))) {
      refreshFailedAt = 0;
    }
    return res;
  },
  async (err: AxiosError) => {
    const originalRequest = err.config as CustomAxiosRequestConfig;

    const url = originalRequest?.url || "";

    const shouldSkip = skipRefreshUrls?.some((path) => url?.includes(path));

    const inCooldown =
      refreshFailedAt > 0 && Date.now() - refreshFailedAt < REFRESH_COOLDOWN_MS;

    if (
      err.response?.status === 401 &&
      originalRequest &&
      !originalRequest._retry &&
      !shouldSkip &&
      !inCooldown
    ) {
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          queue.push({
            resolve: () => resolve(api(originalRequest)),
            reject: (queueErr) => reject(queueErr),
          });
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        await axios.post(`${API_BASE}/auth/refresh`, {}, { withCredentials: true });

        refreshFailedAt = 0;
        isRefreshing = false;
        processQueue(null);

        return api(originalRequest);
      } catch (refreshErr) {
        // Only activate cooldown if the refresh token is explicitly rejected
        // (400 Bad Request or 401 Unauthorized). For network drops or 5xx
        // server issues, leave the cooldown at 0 so we can retry immediately.
        const status = (refreshErr as AxiosError)?.response?.status;
        if (status === 400 || status === 401) {
          refreshFailedAt = Date.now();
        }
        isRefreshing = false;
        processQueue(refreshErr);

        return Promise.reject(refreshErr);
      }
    }

    return Promise.reject(err);
  }
);
