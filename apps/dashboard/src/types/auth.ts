import type { PaginationParams } from "./common";

// === Auth ===
export interface LoginRequest {
  email: string;
  password: string;
  tenant_slug: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
  tenant_name: string;
  tenant_slug: string;
  invite_token?: string;
  license_token?: string;
  checkout_session_id?: string;
}

// === Auth Responses And User Management ===
export interface TokenResponse {
  access_token: string;
  expires_in: number;
  user: User;
  tenant: Tenant;
}

export interface LoginResponse {
  access_token?: string;
  expires_in?: number;
  user?: User;
  tenant?: Tenant;
  requires_2fa?: boolean;
  temp_token?: string;
}

export interface TwoFASetupResponse {
  secret: string;
  qr_url: string;
}

export interface TwoFAStatusResponse {
  enabled: boolean;
  verified_at?: string;
}

export interface CreateUserRequest {
  email: string;
  name: string;
  role: "owner" | "admin" | "member";
  password: string;
}

export interface UpdateUserRequest {
  name?: string;
  role?: "owner" | "admin" | "member";
  role_id?: string;
  language?: string;
}

export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
}

// === Core Models ===
export interface User {
  id: string;
  tenant_id: string;
  email: string;
  name: string;
  role: "owner" | "admin" | "member";
  role_id?: string;
  language?: string | null;
  last_login_at?: string;
  last_logout_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  plan: string;
  settings?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

// === Roles (RBAC) ===
export interface Role {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  is_system: boolean;
  permissions: string[];
  created_at: string;
  updated_at: string;
}

export interface CreateRoleRequest {
  name: string;
  description?: string;
  permissions: string[];
}

export interface UpdateRoleRequest {
  name?: string;
  description?: string;
  permissions?: string[];
}

export type RoleListParams = PaginationParams;

export interface PermissionGroup {
  group: string;
  permissions: string[];
}
