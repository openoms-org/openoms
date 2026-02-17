"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { Integration } from "@/types/api";

export interface ShoperSetupRequest {
  shop_url: string;
  client_id: string;
  client_secret: string;
}

export interface PrestaShopSetupRequest {
  shop_url: string;
  api_key: string;
}

export interface ShopifySetupRequest {
  shop_domain: string;
  access_token: string;
}

export function useSetupShoper() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: ShoperSetupRequest) =>
      apiClient<Integration>("/v1/integrations/shoper/setup", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations"] });
    },
  });
}

export function useSetupPrestaShop() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: PrestaShopSetupRequest) =>
      apiClient<Integration>("/v1/integrations/prestashop/setup", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations"] });
    },
  });
}

export function useSetupShopify() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: ShopifySetupRequest) =>
      apiClient<Integration>("/v1/integrations/shopify/setup", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations"] });
    },
  });
}
