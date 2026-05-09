import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { API_URL, absoluteAPIURL, apiClient, ApiClientError, getErrorMessage, isAuthError } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";

const API_BASE = "*/v1";

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  useAuthStore.getState().clearAuth();
});
afterAll(() => server.close());

describe("apiClient", () => {
  it("uses same-origin API paths by default", async () => {
    expect(API_URL).toBe("");

    let capturedPath = "";
    server.use(
      http.get(`${API_BASE}/orders`, ({ request }) => {
        capturedPath = new URL(request.url).pathname;
        return HttpResponse.json({ items: [], total: 0, limit: 20, offset: 0 });
      })
    );

    const data = await apiClient<{ items: unknown[]; total: number }>("/v1/orders");
    expect(data).toEqual({ items: [], total: 0, limit: 20, offset: 0 });
    expect(capturedPath).toBe("/v1/orders");
  });

  it("builds absolute public URLs from the current origin", () => {
    window.history.pushState(null, "", "/settings/feeds");

    expect(absoluteAPIURL("/v1/feeds/ceneo/t1/token")).toBe(
      `${window.location.origin}/v1/feeds/ceneo/t1/token`
    );
  });

  it("adds Authorization header when token is present", async () => {
    let capturedAuth: string | null = null;

    server.use(
      http.get(`${API_BASE}/orders`, ({ request }) => {
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
      http.get(`${API_BASE}/orders`, ({ request }) => {
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
      http.get(`${API_BASE}/orders`, ({ request }) => {
        capturedContentType = request.headers.get("Content-Type");
        return HttpResponse.json({ items: [] });
      })
    );

    await apiClient("/v1/orders");
    expect(capturedContentType).toBe("application/json");
  });

  it("throws ApiClientError on non-ok response", async () => {
    server.use(
      http.get(`${API_BASE}/orders`, () => {
        return HttpResponse.json({ error: "Not found" }, { status: 404 });
      })
    );

    await expect(apiClient("/v1/orders")).rejects.toThrow(ApiClientError);
    await expect(apiClient("/v1/orders")).rejects.toThrow("Not found");
  });

  it("attempts token refresh on 401 when token exists", async () => {
    let requestCount = 0;

    server.use(
      http.get(`${API_BASE}/orders`, () => {
        requestCount++;
        if (requestCount === 1) {
          return HttpResponse.json({ error: "Unauthorized" }, { status: 401 });
        }
        return HttpResponse.json({ items: [], total: 0 });
      }),
      http.post(`${API_BASE}/auth/refresh`, () => {
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
      http.delete(`${API_BASE}/orders/123`, () => {
        return new HttpResponse(null, { status: 204 });
      })
    );

    const result = await apiClient("/v1/orders/123", { method: "DELETE" });
    expect(result).toBeUndefined();
  });
});

describe("getErrorMessage", () => {
  it("returns server message for 401 when available", () => {
    const err = new ApiClientError(401, "invalid email or password");
    expect(getErrorMessage(err)).toBe("invalid email or password");
  });

  it("returns session expired key for 401 without message", () => {
    const err = new ApiClientError(401, "");
    expect(getErrorMessage(err)).toBe("errors.sessionExpired");
  });

  it("returns rate limit key for 429", () => {
    const err = new ApiClientError(429, "Too many requests");
    expect(getErrorMessage(err)).toBe("errors.tooManyRequests");
  });

  it("returns server error key for 500", () => {
    const err = new ApiClientError(500, "Internal Server Error");
    expect(getErrorMessage(err)).toBe("errors.serverError");
  });

  it("returns the error message for other ApiClientError statuses", () => {
    const err = new ApiClientError(404, "Not found");
    expect(getErrorMessage(err)).toBe("Not found");
  });

  it("returns error message for generic Error", () => {
    const err = new Error("Something broke");
    expect(getErrorMessage(err)).toBe("Something broke");
  });

  it("returns fallback key for non-Error objects", () => {
    expect(getErrorMessage("random string")).toBe("errors.unexpected");
    expect(getErrorMessage(null)).toBe("errors.unexpected");
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
