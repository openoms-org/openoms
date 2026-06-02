import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClientError } from "@/lib/api-client";

const sentryMocks = vi.hoisted(() => ({
  captureException: vi.fn(),
}));

vi.mock("@sentry/nextjs", () => ({
  captureException: sentryMocks.captureException,
}));

import PublicError from "./error";

// next-intl is globally mocked in vitest.setup.ts so useTranslations returns the
// key as-is; assert on the i18n key the component renders.

describe("PublicError", () => {
  beforeEach(() => {
    sentryMocks.captureException.mockClear();
  });

  it("reports unexpected public route errors to Sentry", async () => {
    const error = new Error("public route crashed");

    render(<PublicError error={error} reset={vi.fn()} />);

    await waitFor(() => {
      expect(sentryMocks.captureException).toHaveBeenCalledWith(error);
    });
    expect(screen.getByText("somethingWentWrong")).toBeInTheDocument();
  });

  it("does not report expected public rate-limit errors to Sentry", async () => {
    render(
      <PublicError error={new ApiClientError(429, "Too many requests")} reset={vi.fn()} />,
    );

    await waitFor(() => {
      expect(sentryMocks.captureException).not.toHaveBeenCalled();
    });
  });
});
