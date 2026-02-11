import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  TicketListResponse,
  FreshdeskTicket,
  CreateTicketRequest,
} from "@/types/api";

export function useOrderTickets(orderId: string) {
  return useQuery({
    queryKey: ["helpdesk", "tickets", orderId],
    queryFn: () =>
      apiClient<TicketListResponse>(`/v1/orders/${orderId}/tickets`),
    enabled: !!orderId,
  });
}

export function useAllTickets() {
  return useQuery({
    queryKey: ["helpdesk", "tickets"],
    queryFn: () =>
      apiClient<TicketListResponse>("/v1/helpdesk/tickets"),
  });
}

export function useCreateOrderTicket(orderId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateTicketRequest) =>
      apiClient<FreshdeskTicket>(`/v1/orders/${orderId}/tickets`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["helpdesk", "tickets", orderId] });
      queryClient.invalidateQueries({ queryKey: ["helpdesk", "tickets"] });
    },
  });
}
