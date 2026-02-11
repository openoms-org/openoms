import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from "vitest";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { apiClient, ApiClientError, getErrorMessage, isAuthError } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";

const API_URL = "http://localhost:8080";

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  useAuthStore.getState().clearAuth();
});
afterAll(() => server.close());

describe("apiClient", () => {
  it("makes requests to the correct base URL", async () => {
    server.use(
      http.get(`${API_URL}/v1/orders`, () => {
        return HttpResponse.json({ items: [], total: 0, limit: 20, offset: 0 });
      })
    );

    const data = await apiClient<{ items: unknown[]; total: number }>("/v1/orders");
    expect(data).toEqual({ items: [], total: 0, limit: 20, offset: 0 });
  });

  it("adds Authorization header when token is present", async () => {
    let capturedAuth: string | null = null;

    server.use(
      http.get(`${API_URL}/v1/orders`, ({ request }) => {
        capturedAuth = request.headers.get("Authorization");
        return HttpResponse.json({ items: [] });
      })
    );

    useAuthStore.getState().setAuth("test-token-123", {
      id: "u1",
      tenant_id: "t1",
      email: "test@test.com",
      name: "Test",
      role: "owner",
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    }, {
      id: "t1",
      name: "Test Tenant",
      slug: "test",
      plan: "pro",
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    });

    await apiClient("/v1/orders");
    expect(capturedAuth).toBe("Bearer test-token-123");
  });

  it("does not add Authorization header when no token", async () => {
    let capturedAuth: string | null = null;

    server.use(
      http.get(`${API_URL}/v1/orders`, ({ request }) => {
        capturedAuth = request.headers.get("Authorization");
        return HttpResponse.json({ items: [] });
      })
    );

    await apiClient("/v1/orders");
    expect(capturedAuth).toBeNull();
  });

  it("sets Content-Type to application/json", async () => {
    let capturedContentType: string | null = null;

    server.use(
      http.get(`${API_URL}/v1/orders`, ({ request }) => {
        capturedContentType = request.headers.get("Content-Type");
        return HttpResponse.json({ items: [] });
      })
    );

    await apiClient("/v1/orders");
    expect(capturedContentType).toBe("application/json");
  });

  it("throws ApiClientError on non-ok response", async () => {
    server.use(
      http.get(`${API_URL}/v1/orders`, () => {
        return HttpResponse.json({ error: "Not found" }, { status: 404 });
      })
    );

    await expect(apiClient("/v1/orders")).rejects.toThrow(ApiClientError);
    await expect(apiClient("/v1/orders")).rejects.toThrow("Not found");
  });

  it("attempts token refresh on 401 when token exists", async () => {
    let requestCount = 0;

    server.use(
      http.get(`${API_URL}/v1/orders`, () => {
        requestCount++;
        if (requestCount === 1) {
          return HttpResponse.json({ error: "Unauthorized" }, { status: 401 });
        }
        return HttpResponse.json({ items: [], total: 0 });
      }),
      http.post(`${API_URL}/v1/auth/refresh`, () => {
        return HttpResponse.json({
          access_token: "new-token",
          expires_in: 3600,
          user: {
            id: "u1",
            tenant_id: "t1",
            email: "test@test.com",
            name: "Test",
            role: "owner",
            created_at: "2025-01-01T00:00:00Z",
            updated_at: "2025-01-01T00:00:00Z",
          },
          tenant: {
            id: "t1",
            name: "Test",
            slug: "test",
            plan: "pro",
            created_at: "2025-01-01T00:00:00Z",
            updated_at: "2025-01-01T00:00:00Z",
          },
        });
      })
    );

    useAuthStore.getState().setAuth("old-token", {
      id: "u1",
      tenant_id: "t1",
      email: "test@test.com",
      name: "Test",
      role: "owner",
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    }, {
      id: "t1",
      name: "Test",
      slug: "test",
      plan: "pro",
      created_at: "2025-01-01T00:00:00Z",
      updated_at: "2025-01-01T00:00:00Z",
    });

    const data = await apiClient<{ items: unknown[] }>("/v1/orders");
    expect(data.items).toEqual([]);
    expect(requestCount).toBe(2);
    expect(useAuthStore.getState().token).toBe("new-token");
  });

  it("handles 204 No Content by returning undefined", async () => {
    server.use(
      http.delete(`${API_URL}/v1/orders/123`, () => {
        return new HttpResponse(null, { status: 204 });
      })
    );

    const result = await apiClient("/v1/orders/123", { method: "DELETE" });
    expect(result).toBeUndefined();
  });
});

describe("getErrorMessage", () => {
  it("returns session expired message for 401", () => {
    const err = new ApiClientError(401, "Unauthorized");
    expect(getErrorMessage(err)).toBe("Sesja wygasła. Zaloguj się ponownie.");
  });

  it("returns rate limit message for 429", () => {
    const err = new ApiClientError(429, "Too many requests");
    expect(getErrorMessage(err)).toBe("Zbyt wiele żądań. Poczekaj chwilę i spróbuj ponownie.");
  });

  it("returns server error message for 500", () => {
    const err = new ApiClientError(500, "Internal Server Error");
    expect(getErrorMessage(err)).toBe("Błąd serwera. Spróbuj ponownie później.");
  });

  it("returns the error message for other ApiClientError statuses", () => {
    const err = new ApiClientError(404, "Not found");
    expect(getErrorMessage(err)).toBe("Not found");
  });

  it("returns error message for generic Error", () => {
    const err = new Error("Something broke");
    expect(getErrorMessage(err)).toBe("Something broke");
  });

  it("returns fallback for non-Error objects", () => {
    expect(getErrorMessage("random string")).toBe("Wystąpił nieoczekiwany błąd.");
    expect(getErrorMessage(null)).toBe("Wystąpił nieoczekiwany błąd.");
  });
});

describe("isAuthError", () => {
  it("returns true for 401 ApiClientError", () => {
    expect(isAuthError(new ApiClientError(401, "Unauthorized"))).toBe(true);
  });

  it("returns false for other status codes", () => {
    expect(isAuthError(new ApiClientError(403, "Forbidden"))).toBe(false);
    expect(isAuthError(new ApiClientError(500, "Server Error"))).toBe(false);
  });

  it("returns false for non-ApiClientError", () => {
    expect(isAuthError(new Error("test"))).toBe(false);
    expect(isAuthError(null)).toBe(false);
  });
});
