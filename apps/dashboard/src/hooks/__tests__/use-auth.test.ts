import { describe, it, expect, afterEach } from "vitest";
import { useAuthStore } from "@/lib/auth";
import type { User, Tenant } from "@/types/api";

const mockUser: User = {
  id: "usr-001",
  tenant_id: "t-1",
  email: "admin@example.com",
  name: "Admin User",
  role: "owner",
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

const mockTenant: Tenant = {
  id: "t-1",
  name: "Test Company",
  slug: "test-company",
  plan: "pro",
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

afterEach(() => {
  useAuthStore.getState().clearAuth();
});

describe("useAuthStore", () => {
  it("has correct initial state", () => {
    // clearAuth sets isLoading to false, so we need to check defaults after a fresh module
    const state = useAuthStore.getState();
    expect(state.token).toBeNull();
    expect(state.user).toBeNull();
    expect(state.tenant).toBeNull();
    expect(state.isAuthenticated).toBe(false);
  });

  it("sets auth data via setAuth", () => {
    useAuthStore.getState().setAuth("access-token-123", mockUser, mockTenant);
    const state = useAuthStore.getState();

    expect(state.token).toBe("access-token-123");
    expect(state.user).toEqual(mockUser);
    expect(state.tenant).toEqual(mockTenant);
    expect(state.isAuthenticated).toBe(true);
    expect(state.isLoading).toBe(false);
  });

  it("provides user data after setAuth", () => {
    useAuthStore.getState().setAuth("token", mockUser, mockTenant);
    const user = useAuthStore.getState().user;

    expect(user).not.toBeNull();
    expect(user!.email).toBe("admin@example.com");
    expect(user!.name).toBe("Admin User");
    expect(user!.role).toBe("owner");
  });

  it("provides tenant data after setAuth", () => {
    useAuthStore.getState().setAuth("token", mockUser, mockTenant);
    const tenant = useAuthStore.getState().tenant;

    expect(tenant).not.toBeNull();
    expect(tenant!.name).toBe("Test Company");
    expect(tenant!.slug).toBe("test-company");
    expect(tenant!.plan).toBe("pro");
  });

  it("clears auth data on clearAuth", () => {
    useAuthStore.getState().setAuth("token", mockUser, mockTenant);
    expect(useAuthStore.getState().isAuthenticated).toBe(true);

    useAuthStore.getState().clearAuth();
    const state = useAuthStore.getState();

    expect(state.token).toBeNull();
    expect(state.user).toBeNull();
    expect(state.tenant).toBeNull();
    expect(state.isAuthenticated).toBe(false);
    expect(state.isLoading).toBe(false);
  });

  it("sets loading state via setLoading", () => {
    useAuthStore.getState().setLoading(true);
    expect(useAuthStore.getState().isLoading).toBe(true);

    useAuthStore.getState().setLoading(false);
    expect(useAuthStore.getState().isLoading).toBe(false);
  });

  it("supports multiple setAuth calls (overwriting previous state)", () => {
    useAuthStore.getState().setAuth("token-1", mockUser, mockTenant);
    expect(useAuthStore.getState().token).toBe("token-1");

    const updatedUser = { ...mockUser, name: "Updated User" };
    useAuthStore.getState().setAuth("token-2", updatedUser, mockTenant);
    expect(useAuthStore.getState().token).toBe("token-2");
    expect(useAuthStore.getState().user!.name).toBe("Updated User");
  });
});
