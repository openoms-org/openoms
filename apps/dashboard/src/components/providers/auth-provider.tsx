"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/lib/auth";
import { API_URL } from "@/lib/api-client";
import type { TokenResponse } from "@/types/api";

type RefreshResult =
  | { ok: true; data: TokenResponse }
  | { ok: false; status: number };

// React StrictMode mounts effects twice in development. Sharing the in-flight request
// avoids presenting a rotated refresh cookie twice, which the API treats as token reuse.
let hydrationRefreshPromise: Promise<RefreshResult> | null = null;

function refreshForHydration(): Promise<RefreshResult> {
  if (!hydrationRefreshPromise) {
    hydrationRefreshPromise = fetch(`${API_URL}/v1/auth/refresh`, {
      method: "POST",
      credentials: "include",
    })
      .then(async (res): Promise<RefreshResult> => {
        if (!res.ok) return { ok: false, status: res.status };
        return { ok: true, data: await res.json() as TokenResponse };
      })
      .finally(() => {
        hydrationRefreshPromise = null;
      });
  }
  return hydrationRefreshPromise;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const setAuth = useAuthStore((s) => s.setAuth);
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const setLoading = useAuthStore((s) => s.setLoading);

  useEffect(() => {
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let cancelled = false;

    const hydrate = async () => {
      const secure = typeof window !== "undefined" && window.location.protocol === "https:" ? "; Secure" : "";

      // Skip refresh if no session cookie exists (e.g. on login page)
      const hasSession = document.cookie.split(";").some((c) => c.trim().startsWith("has_session="));
      if (!hasSession) {
        setLoading(false);
        return;
      }

      try {
        const result = await refreshForHydration();
        if (cancelled) return;
        if (result.ok) {
          setAuth(result.data.access_token, result.data.user, result.data.tenant);
          document.cookie = `has_session=1; path=/; SameSite=Lax; max-age=2592000${secure}`;
        } else if (result.status === 429) {
          // Rate-limited — retry after 3s instead of logging out
          retryTimer = setTimeout(() => {
            if (cancelled) return;
            fetch(`${API_URL}/v1/auth/refresh`, { method: "POST", credentials: "include" })
              .then((r) => r.ok ? r.json() : Promise.reject(r))
              .then((data: TokenResponse) => {
                if (cancelled) return;
                setAuth(data.access_token, data.user, data.tenant);
                document.cookie = `has_session=1; path=/; SameSite=Lax; max-age=2592000${secure}`;
              })
              .catch(() => { if (!cancelled) setLoading(false); });
          }, 3000);
        } else {
          clearAuth();
          setLoading(false);
          document.cookie = `has_session=; path=/; max-age=0${secure}`;
        }
      } catch {
        // Network error — keep session if cookie exists, don't log out eagerly
        if (!cancelled) setLoading(false);
      }
    };
    hydrate();

    return () => {
      cancelled = true;
      if (retryTimer) clearTimeout(retryTimer);
    };
  }, [setAuth, clearAuth, setLoading]);

  return <>{children}</>;
}
