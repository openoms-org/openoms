import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type {
  LoyaltyProgram,
  CustomerLoyalty,
  LeaderboardEntry,
  ListResponse,
  LoyaltyProgramListParams,
  CreateLoyaltyProgramRequest,
  UpdateLoyaltyProgramRequest,
  AwardPointsRequest,
  RedeemPointsRequest,
} from "@/types/api";

export function useLoyaltyPrograms(params: LoyaltyProgramListParams = {}) {
  const query = new URLSearchParams();
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.offset != null) query.set("offset", String(params.offset));

  const qs = query.toString();

  return useQuery({
    queryKey: ["loyalty-programs", params],
    queryFn: () =>
      apiClient<ListResponse<LoyaltyProgram>>(
        `/v1/loyalty/programs${qs ? `?${qs}` : ""}`
      ),
  });
}

export function useLoyaltyProgram(id: string) {
  return useQuery({
    queryKey: ["loyalty-programs", id],
    queryFn: () => apiClient<LoyaltyProgram>(`/v1/loyalty/programs/${id}`),
    enabled: !!id,
  });
}

export function useCreateLoyaltyProgram() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateLoyaltyProgramRequest) =>
      apiClient<LoyaltyProgram>("/v1/loyalty/programs", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["loyalty-programs"] });
    },
  });
}

export function useUpdateLoyaltyProgram(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateLoyaltyProgramRequest) =>
      apiClient<LoyaltyProgram>(`/v1/loyalty/programs/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["loyalty-programs"] });
      queryClient.invalidateQueries({ queryKey: ["loyalty-programs", id] });
    },
  });
}

export function useDeleteLoyaltyProgram() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/loyalty/programs/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["loyalty-programs"] });
    },
  });
}

export function useAwardPoints(programId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AwardPointsRequest) =>
      apiClient<{ status: string }>(`/v1/loyalty/programs/${programId}/award`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["loyalty-programs", programId, "leaderboard"],
      });
      queryClient.invalidateQueries({ queryKey: ["customer-loyalty"] });
    },
  });
}

export function useRedeemPoints(programId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: RedeemPointsRequest) =>
      apiClient<{ status: string }>(
        `/v1/loyalty/programs/${programId}/redeem`,
        {
          method: "POST",
          body: JSON.stringify(data),
        }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["loyalty-programs", programId, "leaderboard"],
      });
      queryClient.invalidateQueries({ queryKey: ["customer-loyalty"] });
    },
  });
}

export function useLeaderboard(programId: string, limit = 20) {
  return useQuery({
    queryKey: ["loyalty-programs", programId, "leaderboard", limit],
    queryFn: () =>
      apiClient<LeaderboardEntry[]>(
        `/v1/loyalty/programs/${programId}/leaderboard?limit=${limit}`
      ),
    enabled: !!programId,
  });
}

export function useCustomerLoyaltyStatus(customerId: string) {
  return useQuery({
    queryKey: ["customer-loyalty", customerId],
    queryFn: () =>
      apiClient<CustomerLoyalty[]>(`/v1/loyalty/customers/${customerId}`),
    enabled: !!customerId,
  });
}
