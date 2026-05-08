import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  OAUTH_POPUP_MAX_WAIT_MS,
  OAUTH_POPUP_POLL_MS,
  type OAuthPopupWindow,
  useOAuthPopupMonitor,
} from "./use-oauth-popup-monitor";

function createPopup(open = true): OAuthPopupWindow {
  return {
    closed: !open,
    close: vi.fn(),
  };
}

describe("useOAuthPopupMonitor", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("calls onDone once when the popup closes and clears timers", () => {
    const popup = createPopup();
    const onDone = vi.fn();
    const { result } = renderHook(() => useOAuthPopupMonitor());

    act(() => result.current.start(popup, onDone));
    popup.closed = true;
    act(() => vi.advanceTimersByTime(OAUTH_POPUP_POLL_MS));

    expect(onDone).toHaveBeenCalledTimes(1);
    expect(vi.getTimerCount()).toBe(0);
  });

  it("clears polling on unmount without calling onDone", () => {
    const popup = createPopup();
    const onDone = vi.fn();
    const { result, unmount } = renderHook(() => useOAuthPopupMonitor());

    act(() => result.current.start(popup, onDone));
    unmount();

    popup.closed = true;
    act(() => vi.advanceTimersByTime(OAUTH_POPUP_POLL_MS));

    expect(onDone).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
  });

  it("stops waiting after the maximum OAuth popup duration", () => {
    const popup = createPopup();
    const onDone = vi.fn();
    const { result } = renderHook(() => useOAuthPopupMonitor());

    act(() => result.current.start(popup, onDone));
    act(() => vi.advanceTimersByTime(OAUTH_POPUP_MAX_WAIT_MS));

    expect(onDone).toHaveBeenCalledTimes(1);
    expect(popup.close).toHaveBeenCalledTimes(1);
    expect(vi.getTimerCount()).toBe(0);
  });
});
