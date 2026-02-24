import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { createCrudHooks } from "./create-crud-hooks";
import type {
  Return,
  ReturnListParams,
  CreateReturnRequest,
  UpdateReturnRequest,
  ReturnStatusRequest,
} from "@/types/api";

const returnHooks = createCrudHooks<
  Return,
  CreateReturnRequest,
  UpdateReturnRequest,
  ReturnListParams
>({
  resourceKey: "returns",
  basePath: "/v1/returns",
});

export const useReturns = returnHooks.useList;
export const useReturn = returnHooks.useGet;
export const useCreateReturn = returnHooks.useCreate;
export const useUpdateReturn = returnHooks.useUpdate;
export const useDeleteReturn = returnHooks.useDelete;

export function useTransitionReturnStatus(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: ReturnStatusRequest) =>
      apiClient<Return>(`/v1/returns/${id}/status`, {
        method: "POST",
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["returns"] });
      queryClient.invalidateQueries({ queryKey: ["returns", id] });
    },
  });
}
