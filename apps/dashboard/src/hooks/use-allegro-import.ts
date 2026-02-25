import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

export interface AllegroImportDetail {
  offer_id: string;
  offer_name: string;
  action: "created" | "linked" | "skipped" | "error";
  product_id?: string;
  error?: string;
}

export interface AllegroImportResult {
  total_offers: number;
  created: number;
  linked: number;
  skipped: number;
  errors: number;
  details?: AllegroImportDetail[];
}

export function useImportAllegroOffers() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<AllegroImportResult>(
        "/v1/integrations/allegro/import-offers",
        { method: "POST" }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      queryClient.invalidateQueries({ queryKey: ["allegro", "offers"] });
    },
  });
}
