import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { useAuthStore } from "@/lib/auth";

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

// Import the layout after setting up mocks
import DashboardLayout from "@/app/(dashboard)/layout";

beforeEach(() => {
  useAuthStore.getState().clearAuth();
});

describe("DashboardLayout", () => {
  it("shows loading skeleton when isLoading is true", () => {
    useAuthStore.getState().setLoading(true);

    render(
      <DashboardLayout>
        <div>Child content</div>
      </DashboardLayout>
    );

    // When loading, child content should not be visible
    expect(screen.queryByText("Child content")).not.toBeInTheDocument();
  });

  it("renders children when not loading", () => {
    useAuthStore.getState().setLoading(false);

    render(
      <DashboardLayout>
        <div>Dashboard Content</div>
      </DashboardLayout>
    );

    expect(screen.getByText("Dashboard Content")).toBeInTheDocument();
  });

  it("renders sidebar with OpenOMS branding when not loading", () => {
    useAuthStore.getState().setLoading(false);

    render(
      <DashboardLayout>
        <div>Content</div>
      </DashboardLayout>
    );

    expect(screen.getByText("OpenOMS")).toBeInTheDocument();
  });

  it("renders main content area", () => {
    useAuthStore.getState().setLoading(false);

    const { container } = render(
      <DashboardLayout>
        <div>Main area test</div>
      </DashboardLayout>
    );

    const main = container.querySelector("main");
    expect(main).toBeInTheDocument();
    expect(screen.getByText("Main area test")).toBeInTheDocument();
  });
});
