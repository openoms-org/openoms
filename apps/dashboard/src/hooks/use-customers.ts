import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  Customer,
  ListResponse,
  CustomerListParams,
  CreateCustomerRequest,
  UpdateCustomerRequest,
  Order,
} from "@/types/api";

export function useCustomers(params: CustomerListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));
  if (params.search) query.set("search", params.search);
  if (params.tags) query.set("tags", params.tags);
  if (params.sort_by) query.set("sort_by", params.sort_by);
  if (params.sort_order) query.set("sort_order", params.sort_order);

  const qs = query.toString();

  return useQuery({
    queryKey: ["customers", params],
    queryFn: () =>
      apiClient<ListResponse<Customer>>(`/v1/customers${qs ? `?${qs}` : ""}`),
  });
}

export function useCustomer(id: string) {
  return useQuery({
    queryKey: ["customers", id],
    queryFn: () => apiClient<Customer>(`/v1/customers/${id}`),
    enabled: !!id,
  });
}

export function useCreateCustomer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateCustomerRequest) =>
      apiClient<Customer>("/v1/customers", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["customers"] });
    },
  });
}

export function useUpdateCustomer(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateCustomerRequest) =>
      apiClient<Customer>(`/v1/customers/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["customers"] });
      queryClient.invalidateQueries({ queryKey: ["customers", id] });
    },
  });
}

export function useDeleteCustomer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/customers/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["customers"] });
    },
  });
}

export function useCustomerOrders(customerId: string, params: { limit?: number; offset?: number } = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));

  const qs = query.toString();

  return useQuery({
    queryKey: ["customers", customerId, "orders", params],
    queryFn: () =>
      apiClient<ListResponse<Order>>(
        `/v1/customers/${customerId}/orders${qs ? `?${qs}` : ""}`
      ),
    enabled: !!customerId,
  });
}
