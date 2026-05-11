import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { useAuthStore } from "@/lib/auth";
import type { Tenant, User } from "@/types/api";
import NewMarketplacePage from "./page";

const push = vi.fn();
const replace = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push,
    replace,
  }),
}));

vi.mock("next/link", () => ({
  default: ({ children, href, ...props }: { children: React.ReactNode; href: string }) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock("@/hooks/use-integrations", () => ({
  useCreateIntegration: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}));

const ownerUser: User = {
  id: "user-1",
  tenant_id: "tenant-1",
  email: "owner@example.com",
  name: "Owner",
  role: "owner",
  created_at: "2026-05-11T00:00:00Z",
  updated_at: "2026-05-11T00:00:00Z",
};

const tenant: Tenant = {
  id: "tenant-1",
  name: "OpenOMS",
  slug: "openoms",
  plan: "enterprise",
  created_at: "2026-05-11T00:00:00Z",
  updated_at: "2026-05-11T00:00:00Z",
};

beforeEach(() => {
  push.mockReset();
  replace.mockReset();
  useAuthStore.getState().setAuth("token", ownerUser, tenant);
  useAuthStore.getState().setLoading(false);
});

describe("NewMarketplacePage", () => {
  it("uses Polish diacritics in the provider picker description", () => {
    render(<NewMarketplacePage />);

    expect(
      screen.getByText("Wybierz platformę sprzedażową, z którą chcesz się połączyć"),
    ).toBeInTheDocument();
  });
});
