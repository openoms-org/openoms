"use client";

import { useEffect, useRef, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/lib/auth";
import { apiClient } from "@/lib/api-client";

const INACTIVITY_TIMEOUT_MS = 60 * 60 * 1000; // 60 minutes
const CHECK_INTERVAL_MS = 60 * 1000; // check every 60 seconds
const THROTTLE_MS = 30 * 1000; // update activity timestamp at most every 30 seconds

/**
 * Tracks user activity and expires the session after 60 minutes of inactivity.
 * Idle expiry POSTs /v1/auth/logout (best-effort) so the refresh-token family
 * is revoked server-side, then clears Zustand + the has_session UX cookie.
 * Event listeners are throttled to update the last-activity timestamp at most
 * every 30 seconds. A setInterval check runs every 60 seconds to compare the
 * current time against the last recorded activity.
 */
export function useSessionTimeout() {
  const router = useRouter();
  const lastActivityRef = useRef(0);
  const lastUpdateRef = useRef(0);
  const expiringRef = useRef(false);

  const handleActivity = useCallback(() => {
    const now = Date.now();
    if (now - lastUpdateRef.current >= THROTTLE_MS) {
      lastActivityRef.current = now;
      lastUpdateRef.current = now;
    }
  }, []);

  const expireIdleSession = useCallback(async () => {
    if (expiringRef.current) return;
    expiringRef.current = true;
    await apiClient("/v1/auth/logout", { method: "POST" }).catch(() => {});
    useAuthStore.getState().clearAuth();
    document.cookie = "has_session=; path=/; max-age=0";
    router.push("/login");
  }, [router]);

  useEffect(() => {
    const isAuthenticated = useAuthStore.getState().isAuthenticated;
    if (!isAuthenticated) return;

    // Reset timestamps on mount
    lastActivityRef.current = Date.now();
    lastUpdateRef.current = Date.now();

    const events: Array<keyof WindowEventMap> = ["mousemove", "keydown", "click", "scroll"];
    for (const event of events) {
      window.addEventListener(event, handleActivity, { passive: true });
    }

    const intervalId = setInterval(() => {
      const now = Date.now();
      if (now - lastActivityRef.current >= INACTIVITY_TIMEOUT_MS) {
        void expireIdleSession();
      }
    }, CHECK_INTERVAL_MS);

    return () => {
      for (const event of events) {
        window.removeEventListener(event, handleActivity);
      }
      clearInterval(intervalId);
    };
  }, [handleActivity, expireIdleSession, router]);
}
