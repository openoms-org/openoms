"use client";

import { useRouter } from "next/navigation";
import { useAuthStore } from "@/lib/auth";
import { apiClient } from "@/lib/api-client";
import type { TokenResponse, LoginRequest, RegisterRequest } from "@/types/api";

export function useAuth() {
  const router = useRouter();
  const { user, tenant, isAuthenticated, isLoading } = useAuthStore();

  const login = async (data: LoginRequest) => {
    const res = await apiClient<TokenResponse>("/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });
    useAuthStore.getState().setAuth(res.access_token, res.user, res.tenant);
    document.cookie = "has_session=1; path=/";
    router.push("/");
  };

  const register = async (data: RegisterRequest) => {
    const res = await apiClient<TokenResponse>("/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    });
    useAuthStore.getState().setAuth(res.access_token, res.user, res.tenant);
    document.cookie = "has_session=1; path=/";
    router.push("/");
  };

  const logout = async () => {
    await apiClient("/v1/auth/logout", { method: "POST" }).catch(() => {});
    useAuthStore.getState().clearAuth();
    document.cookie = "has_session=; path=/; max-age=0";
    router.push("/login");
  };

  const isAdmin = user?.role === "admin" || user?.role === "owner";

  return { user, tenant, isAuthenticated, isLoading, isAdmin, login, register, logout };
}
