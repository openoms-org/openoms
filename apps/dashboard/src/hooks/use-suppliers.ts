import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  Supplier,
  ListResponse,
  SupplierListParams,
  CreateSupplierRequest,
  UpdateSupplierRequest,
  SupplierProduct,
  SupplierProductListParams,
  SupplierPortalLinkResponse,
  SupplierPortalStatus,
  SupplierCategoryMapping,
  UpsertCategoryMappingRequest,
  AllegroParameterMapping,
  BulkUpsertAllegroMappingsRequest,
  ImportSupplierProductsRequest,
  ImportSupplierProductsResponse,
  BulkDeleteSupplierProductsRequest,
} from "@/types/api";

export function useSuppliers(params: SupplierListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.status) query.set("status", params.status);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["suppliers", params],
    queryFn: () =>
      apiClient<ListResponse<Supplier>>(`/v1/suppliers${qs ? `?${qs}` : ""}`),
  });
}

export function useSupplier(id: string) {
  return useQuery({
    queryKey: ["suppliers", id],
    queryFn: () => apiClient<Supplier>(`/v1/suppliers/${id}`),
    enabled: !!id,
  });
}

export function useCreateSupplier() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateSupplierRequest) =>
      apiClient<Supplier>("/v1/suppliers", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["suppliers"] });
    },
  });
}

export function useUpdateSupplier(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateSupplierRequest) =>
      apiClient<Supplier>(`/v1/suppliers/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["suppliers"] });
      queryClient.invalidateQueries({ queryKey: ["suppliers", id] });
    },
  });
}

export function useDeleteSupplier() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/suppliers/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["suppliers"] });
    },
  });
}

export function useSyncSupplier() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<{ message: string }>(`/v1/suppliers/${id}/sync`, {
        method: "POST",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["suppliers"] });
    },
  });
}

export function useLinkSupplierProduct(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      supplierProductId,
      productId,
    }: {
      supplierProductId: string;
      productId: string;
    }) =>
      apiClient<SupplierProduct>(
        `/v1/suppliers/${supplierId}/products/${supplierProductId}/link`,
        {
          method: "POST",
          body: JSON.stringify({ product_id: productId }),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["supplier-products", supplierId],
      });
    },
  });
}

export function useImportSupplierProducts(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: ImportSupplierProductsRequest) =>
      apiClient<ImportSupplierProductsResponse>(
        `/v1/suppliers/${supplierId}/products/import`,
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["supplier-products", supplierId],
      });
      queryClient.invalidateQueries({ queryKey: ["products"] });
    },
  });
}

export function useSupplierProducts(
  supplierId: string,
  params: SupplierProductListParams = {}
) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.ean) query.set("ean", params.ean);
  if (params.linked !== undefined) query.set("linked", String(params.linked));
  if (params.search) query.set("search", params.search);
  if (params.category) query.set("category", params.category);

  const qs = query.toString();

  return useQuery({
    queryKey: ["supplier-products", supplierId, params],
    queryFn: () =>
      apiClient<ListResponse<SupplierProduct>>(
        `/v1/suppliers/${supplierId}/products${qs ? `?${qs}` : ""}`
      ),
    enabled: !!supplierId,
  });
}

export function useSupplierSourceCategories(supplierId: string) {
  return useQuery({
    queryKey: ["supplier-source-categories", supplierId],
    queryFn: () =>
      apiClient<string[]>(
        `/v1/suppliers/${supplierId}/products/categories`
      ),
    enabled: !!supplierId,
  });
}

export function useUnlinkSupplierProduct(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (supplierProductId: string) =>
      apiClient<{ message: string }>(
        `/v1/suppliers/${supplierId}/products/${supplierProductId}/unlink`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["supplier-products", supplierId],
      });
    },
  });
}

export function useDeleteSupplierProduct(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (supplierProductId: string) =>
      apiClient<void>(
        `/v1/suppliers/${supplierId}/products/${supplierProductId}`,
        { method: "DELETE" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["supplier-products", supplierId],
      });
    },
  });
}

export function useBulkDeleteSupplierProducts(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: BulkDeleteSupplierProductsRequest) =>
      apiClient<{ deleted: number }>(
        `/v1/suppliers/${supplierId}/products/bulk-delete`,
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["supplier-products", supplierId],
      });
    },
  });
}

// === Supplier Portal (Admin) ===

export function useSupplierPortalStatus(supplierId: string) {
  return useQuery({
    queryKey: ["supplier-portal-status", supplierId],
    queryFn: () =>
      apiClient<SupplierPortalStatus>(
        `/v1/suppliers/${supplierId}/portal/status`
      ),
    enabled: !!supplierId,
  });
}

export function useGeneratePortalLink(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<SupplierPortalLinkResponse>(
        `/v1/suppliers/${supplierId}/portal/generate-link`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["supplier-portal-status", supplierId],
      });
      queryClient.invalidateQueries({ queryKey: ["suppliers", supplierId] });
    },
  });
}

export function useRevokePortalAccess(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<{ message: string }>(
        `/v1/suppliers/${supplierId}/portal/revoke`,
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["supplier-portal-status", supplierId],
      });
      queryClient.invalidateQueries({ queryKey: ["suppliers", supplierId] });
    },
  });
}

// === Supplier Category Mappings ===

export function useSupplierCategoryMappings(supplierId: string) {
  return useQuery({
    queryKey: ["supplier-category-mappings", supplierId],
    queryFn: () =>
      apiClient<SupplierCategoryMapping[]>(
        `/v1/suppliers/${supplierId}/category-mappings`
      ),
    enabled: !!supplierId,
  });
}

export function useUpsertCategoryMapping(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpsertCategoryMappingRequest) =>
      apiClient<SupplierCategoryMapping>(
        `/v1/suppliers/${supplierId}/category-mappings`,
        {
          method: "PUT",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["supplier-category-mappings", supplierId],
      });
    },
  });
}

export function useDeleteCategoryMapping(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (mappingId: string) =>
      apiClient<void>(
        `/v1/suppliers/${supplierId}/category-mappings/${mappingId}`,
        { method: "DELETE" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["supplier-category-mappings", supplierId],
      });
    },
  });
}

// === Allegro Parameter Mappings ===

export function useAllegroParameterMappings(
  supplierId: string,
  allegroCategoryId: string | null
) {
  return useQuery({
    queryKey: ["allegro-parameter-mappings", supplierId, allegroCategoryId],
    queryFn: () =>
      apiClient<AllegroParameterMapping[]>(
        `/v1/suppliers/${supplierId}/allegro-mappings?category_id=${allegroCategoryId}`
      ),
    enabled: !!supplierId && !!allegroCategoryId,
  });
}

export function useBulkUpsertAllegroMappings(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: BulkUpsertAllegroMappingsRequest) =>
      apiClient<AllegroParameterMapping[]>(
        `/v1/suppliers/${supplierId}/allegro-mappings`,
        { method: "PUT", body: JSON.stringify(data) }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro-parameter-mappings", supplierId],
      });
    },
  });
}

export function useDeleteAllegroParameterMapping(supplierId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (mappingId: string) =>
      apiClient<void>(
        `/v1/suppliers/${supplierId}/allegro-mappings/${mappingId}`,
        { method: "DELETE" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["allegro-parameter-mappings", supplierId],
      });
    },
  });
}

export function useSupplierAttributes(supplierId: string) {
  return useQuery({
    queryKey: ["supplier-attributes", supplierId],
    queryFn: () =>
      apiClient<string[]>(`/v1/suppliers/${supplierId}/attributes`),
    enabled: !!supplierId,
  });
}

export function useAllegroMappingCategories(supplierId: string) {
  return useQuery({
    queryKey: ["allegro-mapping-categories", supplierId],
    queryFn: () =>
      apiClient<string[]>(
        `/v1/suppliers/${supplierId}/allegro-mappings/categories`
      ),
    enabled: !!supplierId,
  });
}

export function useProductSupplierLink(productId: string) {
  return useQuery({
    queryKey: ["product-supplier-link", productId],
    queryFn: () =>
      apiClient<{ supplier_id: string | null }>(
        `/v1/products/${productId}/supplier-link`
      ),
    enabled: !!productId,
  });
}
