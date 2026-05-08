import { act, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import AllegroIntegrationPage from "./page";

const refetchIntegrations = vi.fn();
const apiClientMock = vi.fn();
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
    mutate: (_data: unknown, options?: { onSuccess?: () => void }) => {
      options?.onSuccess?.();
    },
  }),
  useUpdateIntegration: () => ({ isPending: false, mutate: vi.fn() }),
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
});
