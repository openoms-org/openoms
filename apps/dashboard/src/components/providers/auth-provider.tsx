"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/lib/auth";
import type { TokenResponse } from "@/types/api";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const setAuth = useAuthStore((s) => s.setAuth);
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const setLoading = useAuthStore((s) => s.setLoading);

  useEffect(() => {
    const hydrate = async () => {
      try {
        const res = await fetch(`${API_URL}/v1/auth/refresh`, {
          method: "POST",
          credentials: "include",
        });
        if (res.ok) {
          const data: TokenResponse = await res.json();
          setAuth(data.access_token, data.user, data.tenant);
          document.cookie = "has_session=1; path=/";
        } else {
          clearAuth();
          document.cookie = "has_session=; path=/; max-age=0";
        }
      } catch {
        clearAuth();
        document.cookie = "has_session=; path=/; max-age=0";
      }
    };
    hydrate();
  }, [setAuth, clearAuth, setLoading]);

  return <>{children}</>;
}
