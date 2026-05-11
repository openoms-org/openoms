import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import OlxIntegrationPage from "./page";

let integrations: unknown[] = [];
const refetchIntegrations = vi.fn();

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

vi.mock("@/hooks/use-integrations", () => ({
  useIntegrations: () => ({
    data: integrations,
    isLoading: false,
    refetch: refetchIntegrations,
  }),
  useCreateIntegration: () => ({ isPending: false, mutate: vi.fn() }),
  useUpdateIntegration: () => ({ isPending: false, mutate: vi.fn() }),
  useDeleteIntegration: () => ({ isPending: false, mutate: vi.fn() }),
}));

vi.mock("@/lib/api-client", () => ({
  apiClient: vi.fn(),
}));

describe("OLX integration error state", () => {
  beforeEach(() => {
    refetchIntegrations.mockClear();
    integrations = [
      {
        id: "int-olx",
        provider: "olx",
        label: "OLX",
        status: "error",
        has_credentials: true,
        settings: {},
        error_message: "OLX authorization expired or was revoked. Reconnect OLX to resume synchronization.",
        created_at: "2026-05-10T00:00:00Z",
        updated_at: "2026-05-10T00:00:00Z",
      },
    ];
  });

  it("keeps the reauthorization action visible when OLX requires reconnect", () => {
    render(<OlxIntegrationPage />);

    expect(screen.getByText("integrationError")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /connectToOlx/i })
    ).toBeInTheDocument();
  });
});
