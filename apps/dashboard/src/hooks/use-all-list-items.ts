import { useQuery, type UseQueryOptions } from "@tanstack/react-query";

import { apiClient } from "@/lib/api-client";
import { buildSearchParams } from "@/lib/search-params";
import type { ListResponse } from "@/types/api";

export const REFERENCE_LIST_PAGE_LIMIT = 100;
const MAX_REFERENCE_LIST_PAGES = 100;

type PageParams<TParams extends object> = TParams & {
  limit: number;
  offset: number;
};

type AllListQueryOptions<TEntity> = Pick<
  UseQueryOptions<ListResponse<TEntity>>,
  "enabled"
>;

export async function fetchAllListItems<
  TEntity,
  TParams extends object = object,
>(
  params: TParams,
  fetchPage: (params: PageParams<TParams>) => Promise<ListResponse<TEntity>>,
): Promise<ListResponse<TEntity>> {
  const items: TEntity[] = [];
  let total = 0;
  let offset = 0;

  for (let page = 0; page < MAX_REFERENCE_LIST_PAGES; page += 1) {
    const response = await fetchPage({
      ...params,
      limit: REFERENCE_LIST_PAGE_LIMIT,
      offset,
    });

    items.push(...response.items);
    total = response.total;

    if (items.length >= total || response.items.length === 0) {
      return {
        items,
        total,
        limit: items.length,
        offset: 0,
      };
    }

    offset = response.offset + response.limit;
  }

  throw new Error(
    `Reference list exceeded ${MAX_REFERENCE_LIST_PAGES * REFERENCE_LIST_PAGE_LIMIT} items`,
  );
}

export function useAllListItems<
  TEntity,
  TParams extends object = object,
>(
  queryKey: readonly unknown[],
  basePath: string,
  params: TParams = {} as TParams,
  queryOptions: AllListQueryOptions<TEntity> = {},
) {
  return useQuery({
    queryKey,
    queryFn: ({ signal }) =>
      fetchAllListItems<TEntity, TParams>(params, (pageParams) => {
        const searchParams = buildSearchParams(
          pageParams as Record<string, string | number | boolean | null | undefined>,
        );
        const query = searchParams.toString();
        return apiClient<ListResponse<TEntity>>(
          `${basePath}${query ? `?${query}` : ""}`,
          { signal },
        );
      }),
    ...queryOptions,
  });
}
