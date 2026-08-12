import { cleanup, render, waitFor } from "@testing-library/react";
import { StrictMode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AuthProvider } from "@/components/providers/auth-provider";
import { useAuthStore } from "@/lib/auth";

function clearCookies() {
  document.cookie.split(";").forEach((cookie) => {
    const name = cookie.split("=")[0]?.trim();
    if (name) {
      document.cookie = `${name}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
    }
  });
}

describe("AuthProvider", () => {
  beforeEach(() => {
    useAuthStore.getState().clearAuth();
    clearCookies();
    document.cookie = "has_session=1; path=/";
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    useAuthStore.getState().clearAuth();
    clearCookies();
  });

  it("deduplicates refresh-token hydration in StrictMode", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        access_token: "access-token",
        user: { id: "user-1", language: "pl" },
        tenant: { id: "tenant-1" },
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <StrictMode>
        <AuthProvider>
          <div>child</div>
        </AuthProvider>
      </StrictMode>,
    );

    await waitFor(() => {
      expect(useAuthStore.getState().isAuthenticated).toBe(true);
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
