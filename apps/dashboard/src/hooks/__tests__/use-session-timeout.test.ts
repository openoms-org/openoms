import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useAuthStore } from "@/lib/auth";

const pushMock = vi.fn();
const apiClientMock = vi.hoisted(() => vi.fn());

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock }),
}));

vi.mock("@/lib/api-client", () => ({
  apiClient: apiClientMock,
}));

import { useSessionTimeout } from "@/hooks/use-session-timeout";

const INACTIVITY_TIMEOUT_MS = 60 * 60 * 1000;
const CHECK_INTERVAL_MS = 60 * 1000;

function expireIdleSession() {
  return act(async () => {
    await vi.advanceTimersByTimeAsync(INACTIVITY_TIMEOUT_MS + CHECK_INTERVAL_MS);
  });
}

describe("useSessionTimeout", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    pushMock.mockReset();
    apiClientMock.mockReset();
    apiClientMock.mockResolvedValue({});
    useAuthStore.getState().clearAuth();
    useAuthStore.setState({ isAuthenticated: true });
    document.cookie = "has_session=1; path=/";
  });

  afterEach(() => {
    vi.useRealTimers();
    useAuthStore.getState().clearAuth();
    document.cookie = "has_session=; path=/; max-age=0";
  });

  it("POSTs /v1/auth/logout on idle timeout so the refresh family is invalidated", async () => {
    renderHook(() => useSessionTimeout());

    await expireIdleSession();

    expect(apiClientMock).toHaveBeenCalledWith("/v1/auth/logout", { method: "POST" });
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(document.cookie).not.toMatch(/has_session=1/);
    expect(pushMock).toHaveBeenCalledWith("/login");
  });

  it("clears local session even when logout POST fails", async () => {
    apiClientMock.mockRejectedValue(new Error("network down"));
    renderHook(() => useSessionTimeout());

    await expireIdleSession();

    expect(apiClientMock).toHaveBeenCalledWith("/v1/auth/logout", { method: "POST" });
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(pushMock).toHaveBeenCalledWith("/login");
  });

  it("does not start the idle timer when the operator is not authenticated", async () => {
    useAuthStore.setState({ isAuthenticated: false });
    renderHook(() => useSessionTimeout());

    await expireIdleSession();

    expect(apiClientMock).not.toHaveBeenCalled();
    expect(pushMock).not.toHaveBeenCalled();
  });
});
