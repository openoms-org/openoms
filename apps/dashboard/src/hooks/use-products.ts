import { createCrudHooks } from "./create-crud-hooks";
import { useAllListItems } from "./use-all-list-items";
import type {
  Product,
  ProductListParams,
  CreateProductRequest,
  UpdateProductRequest,
} from "@/types/api";

const productHooks = createCrudHooks<
  Product,
  CreateProductRequest,
  UpdateProductRequest,
  ProductListParams
>({
  resourceKey: "products",
  basePath: "/v1/products",
});

export const useProducts = productHooks.useList;
export const useProduct = productHooks.useGet;
export const useCreateProduct = productHooks.useCreate;
export const useUpdateProduct = productHooks.useUpdate;
export const useDeleteProduct = productHooks.useDelete;

export function useAllProducts(params: ProductListParams = {}) {
  return useAllListItems<Product, ProductListParams>(
    ["products", "all", params],
    "/v1/products",
    params,
  );
}
