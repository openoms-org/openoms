import { act, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import AllegroIntegrationPage from "./page";

const refetchIntegrations = vi.fn();
const apiClientMock = vi.fn();
const createIntegrationMutate = vi.fn();
const updateIntegrationMutate = vi.fn();
let integrations: unknown[] = [];

vi.mock("next/navigation", () => ({
  useSearchParams: () => new URLSearchParams(),
}));

vi.mock("next/link", () => ({
  default: ({ children, href }: { children: ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

vi.mock("@/components/shared/admin-guard", () => ({
  AdminGuard: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/components/integrations/marketplace-shipment-settings", () => ({
  MarketplaceShipmentSettings: () => null,
}));

vi.mock("@/components/marketplaces/allegro-tab-nav", () => ({
  AllegroTabNav: () => null,
}));

vi.mock("@/hooks/use-allegro", () => ({
  useAllegroAccount: () => ({ data: null, isLoading: false, isError: false }),
}));

vi.mock("@/hooks/use-integrations", () => ({
  useIntegrations: () => ({
    data: integrations,
    isLoading: false,
    refetch: refetchIntegrations,
  }),
  useCreateIntegration: () => ({
    isPending: false,
    mutate: (data: unknown, options?: { onSuccess?: () => void }) => {
      createIntegrationMutate(data);
      options?.onSuccess?.();
    },
  }),
  useUpdateIntegration: () => ({
    isPending: false,
    mutate: (data: unknown, options?: { onSuccess?: () => void }) => {
      updateIntegrationMutate(data);
      options?.onSuccess?.();
    },
  }),
  useDeleteIntegration: () => ({ isPending: false, mutate: vi.fn() }),
}));

vi.mock("@/lib/api-client", () => ({
  apiClient: (...args: unknown[]) => apiClientMock(...args),
}));

describe("Allegro OAuth popup monitoring", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    integrations = [];
    refetchIntegrations.mockClear();
    createIntegrationMutate.mockClear();
    updateIntegrationMutate.mockClear();
    apiClientMock.mockReset();
    apiClientMock.mockResolvedValue({
      auth_url: "https://allegro.example/oauth",
      state: "oauth-state",
      redirect_uri: "https://app.example/marketplaces/allegro",
    });
    vi.spyOn(window, "open").mockReturnValue({
      closed: false,
      close: vi.fn(),
    } as unknown as Window);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it("continues monitoring the OAuth popup after setup state unmounts", async () => {
    const { rerender } = render(<AllegroIntegrationPage />);

    fireEvent.change(screen.getByLabelText("Client ID"), {
      target: { value: "client-id" },
    });
    fireEvent.change(screen.getByLabelText("Client Secret"), {
      target: { value: "client-secret" },
    });
    fireEvent.click(screen.getByRole("button", { name: /zapiszIPrzejdzDoAutoryzacji/i }));

    await act(async () => {
      await Promise.resolve();
    });
    expect(window.open).toHaveBeenCalledTimes(1);
    const popup = vi.mocked(window.open).mock.results[0].value as unknown as {
      closed: boolean;
    };

    integrations = [
      {
        id: "int-allegro",
        provider: "allegro",
        status: "inactive",
        has_credentials: true,
        settings: {},
        created_at: "2026-05-08T00:00:00Z",
        updated_at: "2026-05-08T00:00:00Z",
      },
    ];
    rerender(<AllegroIntegrationPage />);

    popup.closed = true;
    act(() => vi.advanceTimersByTime(500));

    expect(refetchIntegrations).toHaveBeenCalledTimes(1);
  });

  it("sends a tenant-scoped webhook secret when provided during setup", async () => {
    render(<AllegroIntegrationPage />);

    fireEvent.change(screen.getByLabelText("Client ID"), {
      target: { value: "client-id" },
    });
    fireEvent.change(screen.getByLabelText("Client Secret"), {
      target: { value: "client-secret" },
    });
    fireEvent.change(screen.getByLabelText("webhookSecret"), {
      target: { value: "  webhook-secret  " },
    });
    fireEvent.click(screen.getByRole("button", { name: /zapiszIPrzejdzDoAutoryzacji/i }));

    await act(async () => {
      await Promise.resolve();
    });

    expect(createIntegrationMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        credentials: expect.objectContaining({
          client_id: "client-id",
          client_secret: "client-secret",
          webhook_secret: "webhook-secret",
        }),
      })
    );
  });

  it("omits a blank webhook secret during setup", async () => {
    render(<AllegroIntegrationPage />);

    fireEvent.change(screen.getByLabelText("Client ID"), {
      target: { value: "client-id" },
    });
    fireEvent.change(screen.getByLabelText("Client Secret"), {
      target: { value: "client-secret" },
    });
    fireEvent.click(screen.getByRole("button", { name: /zapiszIPrzejdzDoAutoryzacji/i }));

    await act(async () => {
      await Promise.resolve();
    });

    const payload = createIntegrationMutate.mock.calls[0]?.[0] as {
      credentials: Record<string, unknown>;
    };
    expect(payload.credentials).not.toHaveProperty("webhook_secret");
  });

  it("rotates only the tenant-scoped webhook secret for a connected integration", async () => {
    integrations = [
      {
        id: "int-allegro",
        provider: "allegro",
        status: "active",
        has_credentials: true,
        settings: {},
        created_at: "2026-05-08T00:00:00Z",
        updated_at: "2026-05-08T00:00:00Z",
        last_sync_at: "2026-05-08T00:00:00Z",
      },
    ];
    render(<AllegroIntegrationPage />);

    fireEvent.change(screen.getByLabelText("webhookSecret"), {
      target: { value: "  webhook-secret  " },
    });
    fireEvent.click(screen.getByRole("button", { name: /updateCredentials/i }));

    await act(async () => {
      await Promise.resolve();
    });

    expect(updateIntegrationMutate).toHaveBeenCalledWith({
      credentials: {
        sandbox: false,
        webhook_secret: "webhook-secret",
      },
    });
  });
});
