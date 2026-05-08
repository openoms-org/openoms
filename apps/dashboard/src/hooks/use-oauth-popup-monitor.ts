"use client";

import { useCallback, useEffect, useRef } from "react";

export const OAUTH_POPUP_POLL_MS = 500;
export const OAUTH_POPUP_MAX_WAIT_MS = 10 * 60 * 1000;

export type OAuthPopupWindow = {
  closed: boolean;
  close: () => void;
};

type TimerID = ReturnType<typeof setTimeout>;

type OAuthPopupMonitor = {
  start: (popup: OAuthPopupWindow, onDone: () => void) => void;
  clear: () => void;
};

// useOAuthPopupMonitor polls an OAuth popup until it closes and always clears
// the interval/timeout on completion or component unmount.
export function useOAuthPopupMonitor(): OAuthPopupMonitor {
  const intervalRef = useRef<TimerID | null>(null);
  const timeoutRef = useRef<TimerID | null>(null);

  const clear = useCallback(() => {
    if (intervalRef.current !== null) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    if (timeoutRef.current !== null) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
  }, []);

  const start = useCallback(
    (popup: OAuthPopupWindow, onDone: () => void) => {
      clear();
      let completed = false;

      const finish = () => {
        if (completed) {
          return;
        }
        completed = true;
        clear();
        onDone();
      };

      intervalRef.current = setInterval(() => {
        if (popup.closed) {
          finish();
        }
      }, OAUTH_POPUP_POLL_MS);

      timeoutRef.current = setTimeout(() => {
        try {
          popup.close();
        } catch {
          // Ignore cross-browser popup close errors; the monitor still ends.
        }
        finish();
      }, OAUTH_POPUP_MAX_WAIT_MS);
    },
    [clear]
  );

  useEffect(() => clear, [clear]);

  return { start, clear };
}
