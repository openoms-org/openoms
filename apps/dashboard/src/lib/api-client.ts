import { useAuthStore } from "./auth";
import type { TokenResponse, ApiError } from "@/types/api";

const configuredAPIURL = process.env.NEXT_PUBLIC_API_URL?.trim();

export const API_URL = configuredAPIURL ? configuredAPIURL.replace(/\/$/, "") : "";

function apiURL(path: string): string {
  if (/^https?:\/\//i.test(path)) {
    return path;
  }
  return API_URL ? `${API_URL}${path}` : path;
}

export function absoluteAPIURL(path: string): string {
  if (/^https?:\/\//i.test(path)) {
    return path;
  }

  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  const configuredBase = API_URL || "/";
  const base = /^https?:\/\//i.test(configuredBase)
    ? configuredBase
    : typeof window !== "undefined"
      ? new URL(configuredBase, window.location.origin).toString()
      : configuredBase;

  try {
    const baseWithSlash = base.endsWith("/") ? base : `${base}/`;
    return new URL(normalizedPath.replace(/^\//, ""), baseWithSlash).toString();
  } catch {
    return `${base.replace(/\/$/, "")}${normalizedPath}`;
  }
}

function getCSRFToken(): string | null {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : null;
}

let refreshPromise: Promise<string | null> | null = null;

async function refreshToken(): Promise<string | null> {
  try {
    const res = await fetch(apiURL("/v1/auth/refresh"), {
      method: "POST",
      credentials: "include",
    });
    if (!res.ok) {
      useAuthStore.getState().clearAuth();
      return null;
    }
    const data: TokenResponse = await res.json();
    useAuthStore.getState().setAuth(data.access_token, data.user, data.tenant);
    return data.access_token;
  } catch (err) {
    if (process.env.NODE_ENV === "development") {
      console.error("Token refresh failed:", err);
    }
    useAuthStore.getState().clearAuth();
    return null;
  }
}

async function getValidToken(): Promise<string | null> {
  if (refreshPromise) return refreshPromise;
  refreshPromise = refreshToken().finally(() => {
    refreshPromise = null;
  });
  return refreshPromise;
}

export class ApiClientError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "ApiClientError";
  }
}

/**
 * Returns an i18n error key for the given error.
 * Components should translate the returned key via t().
 * For backward compat, returns the server's error.message if available.
 */
export function getErrorMessage(error: unknown): string {
  if (error instanceof ApiClientError) {
    switch (error.status) {
      case 401:
        return error.message || "errors.sessionExpired";
      case 402:
        return error.message || "errors.noActiveSubscription";
      case 403:
        return error.message || "errors.noPermission";
      case 429:
        return "errors.tooManyRequests";
      case 500:
        return "errors.serverError";
      default:
        return error.message || "errors.unexpected";
    }
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "errors.unexpected";
}

/**
 * Returns true if the error is a 401 Unauthorized error.
 */
export function isAuthError(error: unknown): boolean {
  return error instanceof ApiClientError && error.status === 401;
}

/**
 * Handles 402 Payment Required responses by refreshing tenant data
 * in the Zustand store and throwing an ApiClientError.
 */
async function handlePaymentRequired(res: Response): Promise<never> {
  const body = await res.json().catch(() => ({ message: "errors.noActiveSubscription" }));
  const authState = useAuthStore.getState();
  if (authState.isAuthenticated && authState.token) {
    try {
      const meResp = await fetch(apiURL("/v1/users/me"), {
        headers: { Authorization: `Bearer ${authState.token}` },
      });
      if (meResp.ok) {
        const me = await meResp.json();
        authState.setAuth(authState.token, me.user, me.tenant);
      }
    } catch {
      // ignore — banner will show on next navigation
    }
  }
  throw new ApiClientError(402, body.message || "errors.noActiveSubscription");
}

/**
 * Core fetch wrapper with auth, CSRF, 401 auto-refresh, and 402 handling.
 * All public API functions delegate to this.
 */
async function fetchWithAuth(
  url: string,
  init: RequestInit = {},
  extraHeaders: Record<string, string> = {},
): Promise<Response> {
  const token = useAuthStore.getState().token;

  const headers: Record<string, string> = {
    ...extraHeaders,
    ...(init.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const csrfToken = getCSRFToken();
  const method = init.method ?? "GET";
  if (csrfToken && method !== "GET") {
    headers["X-CSRF-Token"] = csrfToken;
  }

  let res = await fetch(url, { ...init, headers, credentials: "include" });

  // Auto-refresh on 401
  if (res.status === 401 && token) {
    const newToken = await getValidToken();
    if (newToken) {
      headers["Authorization"] = `Bearer ${newToken}`;
      res = await fetch(url, { ...init, headers, credentials: "include" });
      if (res.status === 401) {
        useAuthStore.getState().clearAuth();
      }
    }
  }

  if (res.status === 402) {
    await handlePaymentRequired(res);
  }

  return res;
}

export async function apiClient<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const res = await fetchWithAuth(
    apiURL(path),
    options,
    { "Content-Type": "application/json" },
  );

  if (!res.ok) {
    const body: ApiError = await res.json().catch(() => ({ error: "Request failed" }));
    throw new ApiClientError(res.status, body.error);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

export async function apiFetch(
  path: string,
  init?: RequestInit
): Promise<Response> {
  const res = await fetchWithAuth(apiURL(path), init ?? {});

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Request failed" }));
    throw new ApiClientError(res.status, body.error);
  }

  return res;
}

export async function uploadFile(file: File): Promise<{ url: string }> {
  const fd = new FormData();
  fd.append("file", file);

  const res = await fetchWithAuth(apiURL("/v1/uploads"), {
    method: "POST",
    body: fd,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Upload failed" }));
    throw new ApiClientError(res.status, body.error);
  }

  return res.json();
}
