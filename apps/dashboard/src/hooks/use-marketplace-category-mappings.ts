import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  MarketplaceCategoryMapping,
  UpsertMarketplaceCategoryMappingRequest,
} from "@/types/api";

export function useMarketplaceCategoryMappings(integrationId: string) {
  return useQuery({
    queryKey: ["marketplace-category-mappings", integrationId],
    queryFn: () =>
      apiClient<MarketplaceCategoryMapping[]>(
        `/v1/integrations/${integrationId}/category-mappings`
      ),
    enabled: !!integrationId,
  });
}

export function useUpsertMarketplaceCategoryMapping(integrationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpsertMarketplaceCategoryMappingRequest) =>
      apiClient<MarketplaceCategoryMapping>(
        `/v1/integrations/${integrationId}/category-mappings`,
        {
          method: "PUT",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["marketplace-category-mappings", integrationId],
      });
    },
  });
}

export function useDeleteMarketplaceCategoryMapping(integrationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (mappingId: string) =>
      apiClient<void>(
        `/v1/integrations/${integrationId}/category-mappings/${mappingId}`,
        { method: "DELETE" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["marketplace-category-mappings", integrationId],
      });
    },
  });
}
