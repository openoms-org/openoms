import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClientError } from "@/lib/api-client";

const sentryMocks = vi.hoisted(() => ({
  captureException: vi.fn(),
}));

vi.mock("@sentry/nextjs", () => ({
  captureException: sentryMocks.captureException,
}));

import DashboardError from "./error";

describe("DashboardError", () => {
  beforeEach(() => {
    sentryMocks.captureException.mockClear();
  });

  it("reports unexpected dashboard route errors to Sentry", async () => {
    const error = new Error("dashboard crashed");

    render(<DashboardError error={error} reset={vi.fn()} />);

    await waitFor(() => {
      expect(sentryMocks.captureException).toHaveBeenCalledWith(error);
    });
    expect(screen.getByText("Cos poszlo nie tak")).toBeInTheDocument();
  });

  it("does not report expected dashboard auth errors to Sentry", async () => {
    render(
      <DashboardError
        error={new ApiClientError(401, "errors.sessionExpired")}
        reset={vi.fn()}
      />,
    );

    await waitFor(() => {
      expect(sentryMocks.captureException).not.toHaveBeenCalled();
    });
  });
});
