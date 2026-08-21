import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useAuthStore } from "@/lib/auth";
import type { Tenant, User } from "@/types/api";
import SecuritySettingsPage from "./page";

const changePasswordMutate = vi.fn();
const apiClientMock = vi.fn(async (url: string, _init?: unknown) => {
  if (url === "/v1/auth/2fa/status") {
    return { enabled: false };
  }
  if (url === "/v1/api-tokens") {
    return [];
  }
  return {};
});

vi.mock("@/lib/api-client", () => ({
  apiClient: (url: string, ...args: unknown[]) => apiClientMock(url, ...args),
  getErrorMessage: (error: unknown) =>
    error instanceof Error ? error.message : "error",
}));

vi.mock("@/hooks/use-users", () => ({
  useChangePassword: () => ({
    mutate: changePasswordMutate,
    isPending: false,
  }),
}));

vi.mock("@/components/language-selector", () => ({
  LanguageSelector: () => <button type="button">language</button>,
}));

vi.mock("qrcode", () => ({
  default: {
    toCanvas: vi.fn(),
  },
}));

const user: User = {
  id: "user-1",
  tenant_id: "tenant-1",
  email: "owner@example.com",
  name: "Owner",
  role: "owner",
  created_at: "2026-05-14T00:00:00Z",
  updated_at: "2026-05-14T00:00:00Z",
};

const tenant: Tenant = {
  id: "tenant-1",
  name: "OpenOMS",
  slug: "openoms",
  plan: "enterprise",
  created_at: "2026-05-14T00:00:00Z",
  updated_at: "2026-05-14T00:00:00Z",
};

function renderSecurityPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <SecuritySettingsPage />
    </QueryClientProvider>,
  );
}

describe("SecuritySettingsPage password change", () => {
  beforeEach(() => {
    changePasswordMutate.mockReset();
    apiClientMock.mockClear();
    useAuthStore.getState().setAuth("token", user, tenant);
    useAuthStore.getState().setLoading(false);
  });

  it("renders a hidden username field for password managers", async () => {
    renderSecurityPage();

    await screen.findByText("passwordTitle");

    const usernameInput = document.querySelector<HTMLInputElement>(
      'input[name="username"][autocomplete="username"]',
    );
    expect(usernameInput).not.toBeNull();
    expect(usernameInput).toHaveValue("owner@example.com");
  });

  it("submits the password change form when pressing Enter", async () => {
    const userEventApi = userEvent.setup();
    renderSecurityPage();

    await screen.findByText("passwordTitle");

    await userEventApi.type(screen.getByLabelText("currentPassword"), "OldPassword1");
    await userEventApi.type(screen.getByLabelText("newPassword"), "NewPassword1");
    await userEventApi.type(screen.getByLabelText("confirmNewPassword"), "NewPassword1");
    await userEventApi.keyboard("{Enter}");

    expect(changePasswordMutate).toHaveBeenCalledWith(
      {
        current_password: "OldPassword1",
        new_password: "NewPassword1",
      },
      expect.objectContaining({
        onSuccess: expect.any(Function),
        onError: expect.any(Function),
      }),
    );
  });

  it("shows API token create controls for an owner", async () => {
    renderSecurityPage();
    expect(await screen.findByText("apiTokensTitle")).toBeInTheDocument();
    expect(screen.getByLabelText("apiTokensName")).toBeInTheDocument();
  });

  it("hides API token controls for a non-owner", async () => {
    useAuthStore.getState().setAuth(
      "token",
      { ...user, role: "admin" },
      tenant,
    );
    renderSecurityPage();
    await screen.findByText("passwordTitle");
    expect(screen.queryByText("apiTokensTitle")).not.toBeInTheDocument();
  });
});
