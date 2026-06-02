"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";
import { createCrudHooks } from "./create-crud-hooks";
import type {
  Role,
  RoleListParams,
  CreateRoleRequest,
  UpdateRoleRequest,
  PermissionGroup,
} from "@/types/api";

const roleHooks = createCrudHooks<
  Role,
  CreateRoleRequest,
  UpdateRoleRequest,
  RoleListParams
>({
  resourceKey: "roles",
  basePath: "/v1/roles",
});

export const useRoles = roleHooks.useList;
export const useRole = roleHooks.useGet;
export const useCreateRole = roleHooks.useCreate;
export const useUpdateRole = roleHooks.useUpdate;
export const useDeleteRole = roleHooks.useDelete;

export function usePermissionGroups() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return useQuery({
    queryKey: ["permissions"],
    queryFn: () => apiClient<PermissionGroup[]>("/v1/roles/permissions"),
    enabled: isAuthenticated,
  });
}
