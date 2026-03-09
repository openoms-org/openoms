import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

interface EbayPolicy {
  fulfillmentPolicyId?: string;
  returnPolicyId?: string;
  paymentPolicyId?: string;
  name: string;
  marketplaceId: string;
  description?: string;
}

interface EbayPoliciesResponse {
  fulfillment: EbayPolicy[];
  return: EbayPolicy[];
  payment: EbayPolicy[];
}

export function useEbayPolicies(integrationId: string, marketplaceId: string = "EBAY_PL") {
  return useQuery({
    queryKey: ["ebay", "policies", integrationId, marketplaceId],
    queryFn: () =>
      apiClient<EbayPoliciesResponse>(
        `/v1/integrations/ebay/policies?integration_id=${integrationId}&marketplace_id=${marketplaceId}`
      ),
    enabled: !!integrationId,
  });
}

interface CreateEbayListingRequest {
  integration_id: string;
  category_id: string;
  marketplace_id: string;
  fulfillment_policy_id: string;
  return_policy_id: string;
  payment_policy_id?: string;
  condition?: string;
  title?: string;
  description?: string;
  price_override?: number;
  stock_override?: number;
}

export function useCreateEbayListing(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateEbayListingRequest) =>
      apiClient(`/v1/products/${productId}/listings/ebay`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products", productId, "listings"] });
    },
  });
}
