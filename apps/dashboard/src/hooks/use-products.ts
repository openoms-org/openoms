import { createCrudHooks } from "./create-crud-hooks";
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
