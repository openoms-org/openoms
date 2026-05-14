import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiClientError } from "@/lib/api-client";

const sentryMocks = vi.hoisted(() => ({
  captureException: vi.fn(),
}));

vi.mock("@sentry/nextjs", () => ({
  captureException: sentryMocks.captureException,
}));

import { ErrorBoundary } from "./error-boundary";

function ThrowUnexpected() {
  throw new Error("component crashed");
}

function ThrowExpectedAuthError() {
  throw new ApiClientError(401, "errors.sessionExpired");
}

describe("ErrorBoundary", () => {
  beforeEach(() => {
    sentryMocks.captureException.mockClear();
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("reports unexpected component errors to Sentry", () => {
    render(
      <ErrorBoundary>
        <ThrowUnexpected />
      </ErrorBoundary>,
    );

    expect(sentryMocks.captureException).toHaveBeenCalledWith(
      expect.any(Error),
      expect.objectContaining({
        extra: expect.objectContaining({
          componentStack: expect.any(String),
        }),
      }),
    );
    expect(screen.getByText("genericError")).toBeInTheDocument();
  });

  it("does not report expected auth errors to Sentry", () => {
    render(
      <ErrorBoundary>
        <ThrowExpectedAuthError />
      </ErrorBoundary>,
    );

    expect(sentryMocks.captureException).not.toHaveBeenCalled();
    expect(screen.getByText("sessionExpired")).toBeInTheDocument();
  });
});
