import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from "vitest";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import { renderWithProviders, screen, waitFor } from "@/test/test-utils";
import { useAuthStore } from "@/lib/auth";

const API_URL = "http://localhost:8080";

// ---------------------------------------------------------------------------
// Mocks for Next.js navigation
// ---------------------------------------------------------------------------
const mockReplace = vi.fn();

vi.mock("next/navigation", () => ({
  usePathname: vi.fn(() => "/"),
  useRouter: vi.fn(() => ({
    push: vi.fn(),
    replace: mockReplace,
    back: vi.fn(),
  })),
  useSearchParams: vi.fn(() => new URLSearchParams()),
}));

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...props
  }: {
    children: React.ReactNode;
    href: string;
    [key: string]: unknown;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

vi.mock("next-themes", () => ({
  useTheme: vi.fn(() => ({ theme: "light", setTheme: vi.fn() })),
}));

// ---------------------------------------------------------------------------
// Dashboard layout — will check for onboarding redirect
// ---------------------------------------------------------------------------
import DashboardLayout from "@/app/(dashboard)/layout";

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  mockReplace.mockClear();
});
afterAll(() => server.close());

describe("dashboard layout — onboarding redirect", () => {
  it("redirects to /onboarding when onboarding is not completed", async () => {
    useAuthStore.getState().setAuth(
      "test-token",
      { id: "1", name: "Test User", email: "test@test.pl", role: "admin" } as never,
      { id: "1", name: "Test", slug: "test" } as never,
    );

    server.use(
      http.get(`${API_URL}/v1/onboarding/status`, () => {
        return HttpResponse.json({
          completed: false,
          current_step: 1,
          completed_steps: [],
          skipped_steps: [],
          completed_at: null,
          dismissed: false,
        });
      })
    );

    renderWithProviders(
      <DashboardLayout>
        <div>Dashboard content</div>
      </DashboardLayout>
    );

    await waitFor(() => {
      expect(mockReplace).toHaveBeenCalledWith("/onboarding");
    });
  });

  it("does NOT redirect when onboarding is completed", async () => {
    useAuthStore.getState().setAuth(
      "test-token",
      { id: "1", name: "Test User", email: "test@test.pl", role: "admin" } as never,
      { id: "1", name: "Test", slug: "test" } as never,
    );

    server.use(
      http.get(`${API_URL}/v1/onboarding/status`, () => {
        return HttpResponse.json({
          completed: true,
          current_step: 4,
          completed_steps: [1, 2, 3, 4],
          skipped_steps: [],
          completed_at: "2025-03-01T12:00:00Z",
          dismissed: false,
        });
      })
    );

    renderWithProviders(
      <DashboardLayout>
        <div>Dashboard content</div>
      </DashboardLayout>
    );

    // Wait for potential redirect, then verify it didn't happen
    await waitFor(
      () => {
        expect(screen.getByText("Dashboard content")).toBeInTheDocument();
      },
      { timeout: 2000 }
    );

    expect(mockReplace).not.toHaveBeenCalledWith("/onboarding");
  });

  it("does NOT redirect when onboarding is dismissed", async () => {
    useAuthStore.getState().setAuth(
      "test-token",
      { id: "1", name: "Test User", email: "test@test.pl", role: "admin" } as never,
      { id: "1", name: "Test", slug: "test" } as never,
    );

    server.use(
      http.get(`${API_URL}/v1/onboarding/status`, () => {
        return HttpResponse.json({
          completed: false,
          current_step: 1,
          completed_steps: [],
          skipped_steps: [],
          completed_at: null,
          dismissed: true,
        });
      })
    );

    renderWithProviders(
      <DashboardLayout>
        <div>Dashboard content</div>
      </DashboardLayout>
    );

    await waitFor(
      () => {
        expect(screen.getByText("Dashboard content")).toBeInTheDocument();
      },
      { timeout: 2000 }
    );

    expect(mockReplace).not.toHaveBeenCalledWith("/onboarding");
  });
});
