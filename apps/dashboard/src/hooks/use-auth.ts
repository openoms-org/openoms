"use client";

import { useRouter } from "next/navigation";
import { useAuthStore } from "@/lib/auth";
import { apiClient } from "@/lib/api-client";
import type { TokenResponse, LoginRequest, LoginResponse, RegisterRequest } from "@/types/api";

export interface LoginResult {
  requires2FA: boolean;
  tempToken?: string;
}

export function useAuth() {
  const router = useRouter();
  const { user, tenant, isAuthenticated, isLoading } = useAuthStore();

  const login = async (data: LoginRequest): Promise<LoginResult> => {
    const res = await apiClient<LoginResponse>("/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });

    if (res.requires_2fa && res.temp_token) {
      return { requires2FA: true, tempToken: res.temp_token };
    }

    if (res.access_token && res.user && res.tenant) {
      useAuthStore.getState().setAuth(res.access_token, res.user, res.tenant);
      document.cookie = "has_session=1; path=/; SameSite=Lax; max-age=2592000";
      router.push("/");
    }

    return { requires2FA: false };
  };

  const verify2FALogin = async (tempToken: string, code: string) => {
    const res = await apiClient<TokenResponse>("/v1/auth/2fa/login", {
      method: "POST",
      body: JSON.stringify({ temp_token: tempToken, code }),
    });
    useAuthStore.getState().setAuth(res.access_token, res.user, res.tenant);
    document.cookie = "has_session=1; path=/; SameSite=Lax; max-age=2592000";
    router.push("/");
  };

  const register = async (data: RegisterRequest) => {
    const res = await apiClient<TokenResponse>("/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    });
    useAuthStore.getState().setAuth(res.access_token, res.user, res.tenant);
    document.cookie = "has_session=1; path=/; SameSite=Lax; max-age=2592000";
    router.push("/");
  };

  const logout = async () => {
    await apiClient("/v1/auth/logout", { method: "POST" }).catch(() => {});
    useAuthStore.getState().clearAuth();
    document.cookie = "has_session=; path=/; max-age=0";
    router.push("/login");
  };

  const isAdmin = user?.role === "admin" || user?.role === "owner";

  return { user, tenant, isAuthenticated, isLoading, isAdmin, login, verify2FALogin, register, logout };
}
