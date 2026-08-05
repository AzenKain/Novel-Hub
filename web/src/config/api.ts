import axios, { AxiosError, InternalAxiosRequestConfig } from "axios";

export const API_BASE = import.meta.env.VITE_API_BASE_URL || "/api/v1";
export const API_ROOT = API_BASE.replace(/\/api\/v1\/?$/, "");

export function getMediaUrl(path: string): string {
  if (!path) return "";
  if (path.startsWith("http")) return path;
  const cleanPath = path.startsWith("/") ? path : `/${path}`;
  const finalPath = cleanPath.replace(/^\/data\//, '/');
  return `${API_ROOT}${finalPath}`;
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
let authFailed = false;
let queue: QueueItem[] = [];

function getCookie(name: string): string | null {
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(";").shift() || null;
  return null;
}

api.interceptors.request.use((config) => {
  if (config.url?.includes("/auth/signin") || config.url?.includes("/auth/signup")) {
    authFailed = false;
  }
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

api.interceptors.response.use(
  (res) => res,
  async (err: AxiosError) => {
    const originalRequest = err.config as CustomAxiosRequestConfig;

    const url = originalRequest?.url || "";

    const shouldSkip = skipRefreshUrls?.some((path) => url?.includes(path));

    if (
      err.response?.status === 401 &&
      originalRequest &&
      !originalRequest._retry &&
      !shouldSkip &&
      !authFailed
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

        authFailed = false;
        processQueue(null);

        return api(originalRequest);
      } catch (refreshErr) {
        authFailed = true;
        processQueue(refreshErr);
        
        return Promise.reject(refreshErr);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(err);
  }
);
