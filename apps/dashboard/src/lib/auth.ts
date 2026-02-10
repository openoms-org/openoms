import { create } from "zustand";
import type { User, Tenant } from "@/types/api";

interface AuthState {
  token: string | null;
  user: User | null;
  tenant: Tenant | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  setAuth: (token: string, user: User, tenant: Tenant) => void;
  clearAuth: () => void;
  setLoading: (loading: boolean) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  user: null,
  tenant: null,
  isAuthenticated: false,
  isLoading: true,
  setAuth: (token, user, tenant) =>
    set({ token, user, tenant, isAuthenticated: true, isLoading: false }),
  clearAuth: () =>
    set({ token: null, user: null, tenant: null, isAuthenticated: false, isLoading: false }),
  setLoading: (isLoading) => set({ isLoading }),
}));
