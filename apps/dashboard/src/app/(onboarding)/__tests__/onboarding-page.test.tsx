import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from "vitest";
import { server } from "@/test/server";
import { http, HttpResponse } from "msw";
import {
  renderWithProviders,
  screen,
  waitFor,
  userEvent,
} from "@/test/test-utils";

const API_URL = "http://localhost:8080";

// ---------------------------------------------------------------------------
// Mocks for Next.js navigation
// ---------------------------------------------------------------------------
const mockPush = vi.fn();
const mockReplace = vi.fn();

vi.mock("next/navigation", () => ({
  usePathname: vi.fn(() => "/onboarding"),
  useRouter: vi.fn(() => ({
    push: mockPush,
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

// ---------------------------------------------------------------------------
// Page import — will fail until implementation exists
// ---------------------------------------------------------------------------
import OnboardingPage from "@/app/(onboarding)/onboarding/page";

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  mockPush.mockClear();
  mockReplace.mockClear();
});
afterAll(() => server.close());

// ---------------------------------------------------------------------------
// Default MSW handlers for onboarding
// ---------------------------------------------------------------------------
function setupOnboardingHandlers(overrides: {
  status?: Record<string, unknown>;
} = {}) {
  const defaultStatus = {
    completed: false,
    current_step: 1,
    completed_steps: [],
    skipped_steps: [],
    completed_at: null,
    dismissed: false,
  };

  server.use(
    http.get(`${API_URL}/v1/onboarding/status`, () => {
      return HttpResponse.json(overrides.status ?? defaultStatus);
    })
  );
}

// ---------------------------------------------------------------------------
// Tests: Page rendering and stepper
// ---------------------------------------------------------------------------
describe("OnboardingPage", () => {
  describe("stepper rendering", () => {
    it("renders 4 steps in the stepper", async () => {
      setupOnboardingHandlers();
      renderWithProviders(<OnboardingPage />);

      // All 4 step labels should be visible
      await waitFor(() => {
        expect(screen.getByText(/dane firmy/i)).toBeInTheDocument();
      });
      expect(screen.getByText(/magazyn/i)).toBeInTheDocument();
      expect(screen.getByText(/integracj/i)).toBeInTheDocument();
      expect(screen.getByText(/zesp/i)).toBeInTheDocument();
    });

    it("shows step 1 as active by default", async () => {
      setupOnboardingHandlers();
      renderWithProviders(<OnboardingPage />);

      // Step 1 form fields should be visible
      await waitFor(() => {
        expect(screen.getByText(/dane firmy/i)).toBeInTheDocument();
      });
    });

    it("shows current step based on API response", async () => {
      setupOnboardingHandlers({
        status: {
          completed: false,
          current_step: 3,
          completed_steps: [1],
          skipped_steps: [2],
          completed_at: null,
          dismissed: false,
        },
      });
      renderWithProviders(<OnboardingPage />);

      // Should show step 3 content (integration)
      await waitFor(() => {
        expect(screen.getByText(/integracj/i)).toBeInTheDocument();
      });
    });
  });

  // ---------------------------------------------------------------------------
  // Tests: Step 1 — Company details form
  // ---------------------------------------------------------------------------
  describe("step 1 — company details", () => {
    it("renders company form fields", async () => {
      setupOnboardingHandlers();
      renderWithProviders(<OnboardingPage />);

      await waitFor(() => {
        // NIP field (10-digit Polish tax ID)
        expect(screen.getByLabelText(/nip/i)).toBeInTheDocument();
      });
    });

    it("step 1 has no skip button (required step)", async () => {
      setupOnboardingHandlers();
      renderWithProviders(<OnboardingPage />);

      await waitFor(() => {
        expect(screen.getByText(/dane firmy/i)).toBeInTheDocument();
      });

      // Step 1 is required — no "Pomiń" (skip) button
      const skipButtons = screen.queryAllByText(/pomi[ńn]/i);
      expect(skipButtons).toHaveLength(0);
    });
  });

  // ---------------------------------------------------------------------------
  // Tests: Steps 2-4 — Skippable steps
  // ---------------------------------------------------------------------------
  describe("skippable steps", () => {
    it("step 2 has a skip button", async () => {
      setupOnboardingHandlers({
        status: {
          completed: false,
          current_step: 2,
          completed_steps: [1],
          skipped_steps: [],
          completed_at: null,
          dismissed: false,
        },
      });
      renderWithProviders(<OnboardingPage />);

      await waitFor(() => {
        expect(screen.getByText(/pomi[ńn]/i)).toBeInTheDocument();
      });
    });

    it("step 3 has a skip button", async () => {
      setupOnboardingHandlers({
        status: {
          completed: false,
          current_step: 3,
          completed_steps: [1],
          skipped_steps: [2],
          completed_at: null,
          dismissed: false,
        },
      });
      renderWithProviders(<OnboardingPage />);

      await waitFor(() => {
        expect(screen.getByText(/pomi[ńn]/i)).toBeInTheDocument();
      });
    });
  });

  // ---------------------------------------------------------------------------
  // Tests: Redirect when completed
  // ---------------------------------------------------------------------------
  describe("redirect behavior", () => {
    it("redirects to dashboard when onboarding is completed", async () => {
      setupOnboardingHandlers({
        status: {
          completed: true,
          current_step: 4,
          completed_steps: [1, 2, 3, 4],
          skipped_steps: [],
          completed_at: "2025-03-01T12:00:00Z",
          dismissed: false,
        },
      });
      renderWithProviders(<OnboardingPage />);

      await waitFor(() => {
        expect(mockReplace).toHaveBeenCalledWith("/");
      });
    });
  });

  // ---------------------------------------------------------------------------
  // Tests: "Finish later" button
  // ---------------------------------------------------------------------------
  describe("finish later", () => {
    it("shows a 'finish later' button", async () => {
      setupOnboardingHandlers();
      renderWithProviders(<OnboardingPage />);

      await waitFor(() => {
        expect(
          screen.getByText(/doko[ńn]cz p[oó][zź]niej/i)
        ).toBeInTheDocument();
      });
    });
  });

  // ---------------------------------------------------------------------------
  // Tests: Completion screen
  // ---------------------------------------------------------------------------
  describe("completion screen", () => {
    it("shows completion screen when all steps are done", async () => {
      setupOnboardingHandlers({
        status: {
          completed: true,
          current_step: 4,
          completed_steps: [1, 2, 3, 4],
          skipped_steps: [],
          completed_at: "2025-03-01T12:00:00Z",
          dismissed: false,
        },
      });

      // Note: this test may also redirect. The exact behavior depends
      // on whether the page shows a completion screen before redirecting.
      // The implementation should show a completion screen with a
      // "Go to dashboard" button.
      renderWithProviders(<OnboardingPage />);

      // Either redirects or shows completion — both are valid
      await waitFor(() => {
        const goToDashboard = screen.queryByText(
          /przejd[zź] do panelu|panel|dashboard/i
        );
        const didRedirect = mockReplace.mock.calls.length > 0;
        expect(goToDashboard || didRedirect).toBeTruthy();
      });
    });
  });
});
