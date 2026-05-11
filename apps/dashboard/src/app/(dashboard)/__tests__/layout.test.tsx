import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useAuthStore } from "@/lib/auth";
import type { Tenant, User } from "@/types/api";

// Mock next/navigation which is used by the sidebar
vi.mock("next/navigation", () => ({
  usePathname: vi.fn(() => "/"),
  useRouter: vi.fn(() => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
  })),
  useSearchParams: vi.fn(() => new URLSearchParams()),
}));

// Mock next/link to render a simple anchor
vi.mock("next/link", () => ({
  default: ({ children, href, ...props }: { children: React.ReactNode; href: string }) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

// Mock next-themes
vi.mock("next-themes", () => ({
  useTheme: vi.fn(() => ({ theme: "light", setTheme: vi.fn() })),
}));

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: vi.fn(() => (key: string) => key),
}));

vi.mock("@/hooks/use-onboarding-wizard", () => ({
  useOnboardingStatus: vi.fn(() => ({
    data: { completed: true, dismissed: false },
  })),
}));

vi.mock("@/components/subscription-banner", () => ({
  SubscriptionBanner: () => null,
}));

vi.mock("@/hooks/use-websocket", () => ({
  useWebSocket: vi.fn(() => ({ isConnected: false, lastEvent: null })),
}));

vi.mock("@/hooks/use-service-worker", () => ({
  useServiceWorker: vi.fn(),
}));

// Ensure localStorage is available (happy-dom may lag behind)
if (typeof globalThis.localStorage === "undefined" || typeof globalThis.localStorage.getItem !== "function") {
  const store: Record<string, string> = {};
  globalThis.localStorage = {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => { store[key] = value; },
    removeItem: (key: string) => { delete store[key]; },
    clear: () => { Object.keys(store).forEach((k) => delete store[k]); },
    key: (index: number) => Object.keys(store)[index] ?? null,
    length: 0,
  } as Storage;
}

// Import the layout after setting up mocks
import DashboardLayout from "@/app/(dashboard)/layout";

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

function Wrapper({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}

beforeEach(() => {
  useAuthStore.getState().clearAuth();
  localStorage.clear();
});

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

describe("DashboardLayout", () => {
  it("shows loading skeleton when isLoading is true", () => {
    useAuthStore.getState().setLoading(true);

    render(
      <DashboardLayout>
        <div>Child content</div>
      </DashboardLayout>,
      { wrapper: Wrapper },
    );

    // When loading, child content should not be visible
    expect(screen.queryByText("Child content")).not.toBeInTheDocument();
  });

  it("renders children when not loading", () => {
    useAuthStore.getState().setLoading(false);

    render(
      <DashboardLayout>
        <div>Dashboard Content</div>
      </DashboardLayout>,
      { wrapper: Wrapper },
    );

    expect(screen.getByText("Dashboard Content")).toBeInTheDocument();
  });

  it("renders sidebar with OpenOMS branding when not loading", () => {
    useAuthStore.getState().setLoading(false);

    render(
      <DashboardLayout>
        <div>Content</div>
      </DashboardLayout>,
      { wrapper: Wrapper },
    );

    expect(screen.getByText("OpenOMS")).toBeInTheDocument();
  });

  it("gives the collapsed home link an accessible name", () => {
    useAuthStore.getState().setAuth("token", ownerUser, tenant);
    localStorage.setItem("sidebar-collapsed", "true");

    render(
      <DashboardLayout>
        <div>Content</div>
      </DashboardLayout>,
      { wrapper: Wrapper },
    );

    expect(
      screen.getAllByRole("link", { name: "operationsDashboard" }).length,
    ).toBeGreaterThan(0);
  });

  it("renders main content area", () => {
    useAuthStore.getState().setLoading(false);

    const { container } = render(
      <DashboardLayout>
        <div>Main area test</div>
      </DashboardLayout>,
      { wrapper: Wrapper },
    );

    const main = container.querySelector("main");
    expect(main).toBeInTheDocument();
    expect(screen.getByText("Main area test")).toBeInTheDocument();
  });

  it("hides non-ready items from the expanded client-ready sales navigation", () => {
    useAuthStore.getState().setAuth("token", ownerUser, tenant);

    render(
      <DashboardLayout>
        <div>Content</div>
      </DashboardLayout>,
      { wrapper: Wrapper },
    );

    expect(screen.getByText("orders")).toBeInTheDocument();
    expect(screen.getByText("customers")).toBeInTheDocument();
    expect(screen.getByText("returns")).toBeInTheDocument();

    expect(screen.queryByText("invoices")).not.toBeInTheDocument();
    expect(screen.queryByText("invoicing")).not.toBeInTheDocument();
  });

  it("shows the client-ready courier module in logistics navigation", async () => {
    useAuthStore.getState().setAuth("token", ownerUser, tenant);
    localStorage.setItem("sidebar-expanded-groups", JSON.stringify(["sales", "logistics"]));

    render(
      <DashboardLayout>
        <div>Content</div>
      </DashboardLayout>,
      { wrapper: Wrapper },
    );

    expect(await screen.findByText("shipments")).toBeInTheDocument();
    expect(await screen.findByText("carriers")).toBeInTheDocument();

    expect(screen.queryByText("packing")).not.toBeInTheDocument();
    expect(screen.queryByText("pickPack")).not.toBeInTheDocument();
    expect(screen.queryByText("stocktakes")).not.toBeInTheDocument();
  });
});
