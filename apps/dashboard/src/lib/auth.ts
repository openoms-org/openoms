import { create } from "zustand";
import type { User, Tenant } from "@/types/api";

interface AuthState {
  token: string | null;
  user: User | null;
  tenant: Tenant | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  locale: string;
  setAuth: (token: string, user: User, tenant: Tenant) => void;
  clearAuth: () => void;
  setLoading: (loading: boolean) => void;
  setLocale: (locale: string) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  user: null,
  tenant: null,
  isAuthenticated: false,
  isLoading: true,
  locale: "en",
  setAuth: (token, user, tenant) =>
    set({
      token,
      user,
      tenant,
      isAuthenticated: true,
      isLoading: false,
      ...(user.language ? { locale: user.language } : {}),
    }),
  clearAuth: () =>
    set({
      token: null,
      user: null,
      tenant: null,
      isAuthenticated: false,
      isLoading: false,
      locale: "en",
    }),
  setLoading: (isLoading) => set({ isLoading }),
  setLocale: (locale) => set({ locale }),
}));
