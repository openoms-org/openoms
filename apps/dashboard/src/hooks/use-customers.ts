import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import { createCrudHooks } from "./create-crud-hooks";
import type {
  Customer,
  ListResponse,
  CustomerListParams,
  CreateCustomerRequest,
  UpdateCustomerRequest,
  Order,
} from "@/types/api";

const customerHooks = createCrudHooks<
  Customer,
  CreateCustomerRequest,
  UpdateCustomerRequest,
  CustomerListParams
>({
  resourceKey: "customers",
  basePath: "/v1/customers",
});

export const useCustomers = customerHooks.useList;
export const useCustomer = customerHooks.useGet;
export const useCreateCustomer = customerHooks.useCreate;
export const useUpdateCustomer = customerHooks.useUpdate;
export const useDeleteCustomer = customerHooks.useDelete;

export function useCustomerOrders(customerId: string, params: { limit?: number; offset?: number } = {}) {
  const sp = buildSearchParams(params);
  const qs = sp.toString();

  return useQuery({
    queryKey: ["customers", customerId, "orders", params],
    queryFn: () =>
      apiClient<ListResponse<Order>>(
        `/v1/customers/${customerId}/orders${qs ? `?${qs}` : ""}`
      ),
    enabled: !!customerId,
  });
}
