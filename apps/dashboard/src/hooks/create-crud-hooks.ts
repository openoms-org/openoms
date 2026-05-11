import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import type { ListResponse } from "@/types/api";

interface CrudHooksOptions {
  /** React Query key prefix, e.g. "customers" */
  resourceKey: string;
  /** API path prefix, e.g. "/v1/customers" */
  basePath: string;
  /** HTTP method for updates, defaults to "PATCH" */
  updateMethod?: "PATCH" | "PUT";
}

/**
 * Creates a standard set of CRUD hooks for a resource.
 *
 * Usage:
 * ```ts
 * const { useList, useGet, useCreate, useUpdate, useDelete } = createCrudHooks<
 *   Customer, CreateCustomerRequest, UpdateCustomerRequest, CustomerListParams
 * >({ resourceKey: "customers", basePath: "/v1/customers" });
 * ```
 */
 
export function createCrudHooks<
  TEntity,
  TCreate = Partial<TEntity>,
  TUpdate = Partial<TEntity>,
  TParams extends object = object,
>(options: CrudHooksOptions) {
  const { resourceKey, basePath, updateMethod = "PATCH" } = options;

  function useList(params: TParams = {} as TParams) {
    const sp = buildSearchParams(params);
    const qs = sp.toString();

    return useQuery({
      queryKey: [resourceKey, params],
      queryFn: () =>
        apiClient<ListResponse<TEntity>>(
          `${basePath}${qs ? `?${qs}` : ""}`
        ),
    });
  }

  function useGet(id: string) {
    return useQuery({
      queryKey: [resourceKey, id],
      queryFn: () => apiClient<TEntity>(`${basePath}/${id}`),
      enabled: !!id,
    });
  }

  function useCreate() {
    const queryClient = useQueryClient();
    return useMutation({
      mutationFn: (data: TCreate) =>
        apiClient<TEntity>(basePath, {
          method: "POST",
          body: JSON.stringify(data),
        }),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: [resourceKey] });
      },
    });
  }

  function useUpdate(id: string) {
    const queryClient = useQueryClient();
    return useMutation({
      mutationFn: (data: TUpdate) =>
        apiClient<TEntity>(`${basePath}/${id}`, {
          method: updateMethod,
          body: JSON.stringify(data),
        }),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: [resourceKey] });
        queryClient.invalidateQueries({ queryKey: [resourceKey, id] });
      },
    });
  }

  function useDelete() {
    const queryClient = useQueryClient();
    return useMutation({
      mutationFn: (deleteId: string) =>
        apiClient<void>(`${basePath}/${deleteId}`, { method: "DELETE" }),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: [resourceKey] });
      },
    });
  }

  return { useList, useGet, useCreate, useUpdate, useDelete };
}
