import { useAuthStore } from "./auth";
import type { TokenResponse, ApiError } from "@/types/api";

export const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

function getCSRFToken(): string | null {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : null;
}

let refreshPromise: Promise<string | null> | null = null;

async function refreshToken(): Promise<string | null> {
  try {
    const res = await fetch(`${API_URL}/v1/auth/refresh`, {
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
  refreshPromise = refreshToken();
  try {
    return await refreshPromise;
  } finally {
    refreshPromise = null;
  }
}

export class ApiClientError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "ApiClientError";
  }
}

/**
 * Returns a user-friendly error message based on the HTTP status code.
 */
/**
 * Returns an i18n error key for the given error.
 * Components should translate the returned key via t().
 * For backward compat, returns the server's error.message if available.
 */
export function getErrorMessage(error: unknown): string {
  if (error instanceof ApiClientError) {
    switch (error.status) {
      case 401:
        return "errors.sessionExpired";
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
      const meResp = await fetch(`${API_URL}/v1/users/me`, {
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

export async function apiClient<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = useAuthStore.getState().token;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const csrfToken = getCSRFToken();
  if (csrfToken && options.method && options.method !== "GET") {
    headers["X-CSRF-Token"] = csrfToken;
  }

  let res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers,
    credentials: "include",
  });

  // Auto-refresh on 401
  if (res.status === 401 && token) {
    const newToken = await getValidToken();
    if (newToken) {
      headers["Authorization"] = `Bearer ${newToken}`;
      res = await fetch(`${API_URL}${path}`, {
        ...options,
        headers,
        credentials: "include",
      });
      // If still 401 after refresh, clear auth and throw
      if (res.status === 401) {
        useAuthStore.getState().clearAuth();
      }
    }
  }

  if (res.status === 402) {
    await handlePaymentRequired(res);
  }

  if (!res.ok) {
    const body: ApiError = await res.json().catch(() => ({ error: "Request failed" }));
    throw new ApiClientError(res.status, body.error);
  }

  // Handle 204 No Content
  if (res.status === 204) return undefined as T;

  return res.json();
}

export async function apiFetch(
  path: string,
  init?: RequestInit
): Promise<Response> {
  const token = useAuthStore.getState().token;

  const headers: Record<string, string> = {
    ...(init?.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const csrfToken = getCSRFToken();
  if (csrfToken && init?.method && init.method !== "GET") {
    headers["X-CSRF-Token"] = csrfToken;
  }

  let res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers,
    credentials: "include",
  });

  // Auto-refresh on 401
  if (res.status === 401 && token) {
    const newToken = await getValidToken();
    if (newToken) {
      headers["Authorization"] = `Bearer ${newToken}`;
      res = await fetch(`${API_URL}${path}`, {
        ...init,
        headers,
        credentials: "include",
      });
      if (res.status === 401) {
        useAuthStore.getState().clearAuth();
      }
    }
  }

  if (res.status === 402) {
    await handlePaymentRequired(res);
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Request failed" }));
    throw new ApiClientError(res.status, body.error);
  }

  return res;
}

export async function uploadFile(file: File): Promise<{ url: string }> {
  const token = useAuthStore.getState().token;

  function buildFormData(): FormData {
    const fd = new FormData();
    fd.append("file", file);
    return fd;
  }

  const headers: Record<string, string> = {};
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const csrfToken = getCSRFToken();
  if (csrfToken) {
    headers["X-CSRF-Token"] = csrfToken;
  }

  let res = await fetch(`${API_URL}/v1/uploads`, {
    method: "POST",
    headers,
    body: buildFormData(),
    credentials: "include",
  });

  // Auto-refresh on 401
  if (res.status === 401 && token) {
    const newToken = await getValidToken();
    if (newToken) {
      headers["Authorization"] = `Bearer ${newToken}`;
      res = await fetch(`${API_URL}/v1/uploads`, {
        method: "POST",
        headers,
        body: buildFormData(),
        credentials: "include",
      });
    }
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Upload failed" }));
    throw new ApiClientError(res.status, body.error);
  }

  return res.json();
}
