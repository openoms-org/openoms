"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/lib/auth";
import { API_URL } from "@/lib/api-client";
import type { TokenResponse } from "@/types/api";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const setAuth = useAuthStore((s) => s.setAuth);
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const setLoading = useAuthStore((s) => s.setLoading);

  useEffect(() => {
    const hydrate = async () => {
      const secure = typeof window !== "undefined" && window.location.protocol === "https:" ? "; Secure" : "";

      // Skip refresh if no session cookie exists (e.g. on login page)
      const hasSession = document.cookie.split(";").some((c) => c.trim().startsWith("has_session="));
      if (!hasSession) {
        setLoading(false);
        return;
      }

      try {
        const res = await fetch(`${API_URL}/v1/auth/refresh`, {
          method: "POST",
          credentials: "include",
        });
        if (res.ok) {
          const data: TokenResponse = await res.json();
          setAuth(data.access_token, data.user, data.tenant);
          document.cookie = `has_session=1; path=/; SameSite=Lax; max-age=2592000${secure}`;
        } else if (res.status === 429) {
          // Rate-limited — retry after 3s instead of logging out
          setTimeout(() => {
            fetch(`${API_URL}/v1/auth/refresh`, { method: "POST", credentials: "include" })
              .then((r) => r.ok ? r.json() : Promise.reject(r))
              .then((data: TokenResponse) => {
                setAuth(data.access_token, data.user, data.tenant);
                document.cookie = `has_session=1; path=/; SameSite=Lax; max-age=2592000${secure}`;
              })
              .catch(() => setLoading(false));
          }, 3000);
        } else {
          clearAuth();
          setLoading(false);
          document.cookie = `has_session=; path=/; max-age=0${secure}`;
        }
      } catch {
        // Network error — keep session if cookie exists, don't log out eagerly
        setLoading(false);
      }
    };
    hydrate();
  }, [setAuth, clearAuth, setLoading]);

  return <>{children}</>;
}
