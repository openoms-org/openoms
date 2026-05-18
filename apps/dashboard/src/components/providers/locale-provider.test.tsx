import { act, cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { LocaleProvider } from "@/components/providers/locale-provider";
import { useAuthStore } from "@/lib/auth";
import type { Tenant, User } from "@/types/api";

const mockTenant: Tenant = {
  id: "tenant-1",
  name: "OpenOMS",
  slug: "openoms",
  plan: "enterprise",
  created_at: "2026-05-18T00:00:00Z",
  updated_at: "2026-05-18T00:00:00Z",
};

const mockUser: User = {
  id: "user-1",
  tenant_id: "tenant-1",
  email: "owner@example.com",
  name: "Owner",
  role: "owner",
  language: "pl",
  created_at: "2026-05-18T00:00:00Z",
  updated_at: "2026-05-18T00:00:00Z",
};

function clearCookies() {
  document.cookie.split(";").forEach((cookie) => {
    const name = cookie.split("=")[0]?.trim();
    if (name) {
      document.cookie = `${name}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
    }
  });
}

describe("LocaleProvider", () => {
  let reloadSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    useAuthStore.getState().clearAuth();
    sessionStorage.clear();
    clearCookies();
    document.cookie = "NEXT_LOCALE=en; path=/";
    reloadSpy = vi.spyOn(window.location, "reload").mockImplementation(() => undefined);
  });

  afterEach(() => {
    cleanup();
    reloadSpy.mockRestore();
    useAuthStore.getState().clearAuth();
    sessionStorage.clear();
    clearCookies();
  });

  it("syncs hydrated user language to the locale cookie without hard reload", async () => {
    act(() => {
      useAuthStore.getState().setAuth("token", mockUser, mockTenant);
    });

    render(
      <LocaleProvider>
        <div>child</div>
      </LocaleProvider>,
    );

    await waitFor(() => {
      expect(document.cookie).toContain("NEXT_LOCALE=pl");
    });
    expect(reloadSpy).not.toHaveBeenCalled();
  });
});
