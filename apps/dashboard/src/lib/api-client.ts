import { useAuthStore } from "./auth";
import type { TokenResponse, ApiError } from "@/types/api";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

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
export function getErrorMessage(error: unknown): string {
  if (error instanceof ApiClientError) {
    switch (error.status) {
      case 401:
        return "Sesja wygasła. Zaloguj się ponownie.";
      case 429:
        return "Zbyt wiele żądań. Poczekaj chwilę i spróbuj ponownie.";
      case 500:
        return "Błąd serwera. Spróbuj ponownie później.";
      default:
        return error.message || "Wystąpił nieoczekiwany błąd.";
    }
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "Wystąpił nieoczekiwany błąd.";
}

/**
 * Returns true if the error is a 401 Unauthorized error.
 */
export function isAuthError(error: unknown): boolean {
  return error instanceof ApiClientError && error.status === 401;
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

  if (!res.ok) {
    const body: ApiError = await res.json().catch(() => ({ error: "Request failed" }));
    throw new ApiClientError(res.status, body.error);
  }

  // Handle 204 No Content
  if (res.status === 204) return undefined as T;

  return res.json();
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
