"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { FormWrapper } from "@/components/shared/form-wrapper";
import { TagInput } from "@/components/shared/tag-input";
import { Separator } from "@/components/ui/separator";
import { Label } from "@/components/ui/label";
import type { Product, CreateProductRequest } from "@/types/api";
import { normalizeProductImages } from "@/types/api";
import { ProductBasicFields } from "./product-basic-fields";
import { ProductDescriptionFields } from "./product-description-fields";
import { ProductDimensionsFields } from "./product-dimensions-fields";
import { ProductDropshipFields } from "./product-dropship-fields";
import { ProductImagesFields } from "./product-images-fields";
import {
  createProductSchema,
  type ProductFormValues,
} from "./product-form.schema";

interface ProductFormProps {
  product?: Product;
  onSubmit: (data: CreateProductRequest) => void;
  isLoading?: boolean;
}

export function ProductForm({ product, onSubmit, isLoading }: ProductFormProps) {
  const t = useTranslations("products");
  const tv = useTranslations("products.validation");
  const tc = useTranslations("common");
  const [imageList, setImageList] = useState<{ url: string; alt: string }[]>(
    normalizeProductImages(product?.images).map((img) => ({ url: img.url, alt: img.alt || "" }))
  );
  const [tags, setTags] = useState<string[]>(product?.tags || []);
  const [selectedCategoryId, setSelectedCategoryId] = useState<string | undefined>(
    product?.category_id || undefined
  );
  const [isDropship, setIsDropship] = useState(product?.is_dropship || false);
  const [dropshipSupplierId, setDropshipSupplierId] = useState<string>(
    product?.dropship_supplier_id || ""
  );
  const productSchema = useMemo(() => createProductSchema(tv), [tv]);

  const handleFormSubmit = (data: ProductFormValues) => {
    onSubmit({
      ...data,
      description_short: data.description_short || undefined,
      description_long: data.description_long || undefined,
      weight: isNaN(data.weight as number) ? undefined : data.weight,
      width: isNaN(data.width as number) ? undefined : data.width,
      height: isNaN(data.height as number) ? undefined : data.height,
      depth: isNaN(data.depth as number) ? undefined : data.depth,
      image_url: data.image_url || undefined,
      images: imageList
        .filter((img) => img.url.trim() !== "")
        .map((img, i) => ({ url: img.url, alt: img.alt || undefined, position: i + 1 })),
      tags: tags.length > 0 ? tags : undefined,
      category_id: selectedCategoryId || undefined,
      is_dropship: isDropship || undefined,
      dropship_supplier_id: isDropship && dropshipSupplierId ? dropshipSupplierId : undefined,
    });
  };

  const addImage = () => {
    if (imageList.length >= 16) {
      toast.error(t("form.maxImagesReached"));
      return;
    }
    setImageList([...imageList, { url: "", alt: "" }]);
  };

  const removeImage = (index: number) => {
    setImageList(imageList.filter((_, i) => i !== index));
  };

  const updateImage = (index: number, field: "url" | "alt", value: string) => {
    setImageList(
      imageList.map((img, i) => (i === index ? { ...img, [field]: value } : img))
    );
  };

  return (
    <FormWrapper<ProductFormValues>
      schema={productSchema}
      defaultValues={{
        name: product?.name ?? "",
        sku: product?.sku ?? "",
        ean: product?.ean ?? "",
        price: product?.price ?? 0,
        stock_quantity: product?.stock_quantity ?? 0,
        source: (product?.source as ProductFormValues["source"]) ?? "manual",
        description_short: product?.description_short || "",
        description_long: product?.description_long || "",
        weight: product?.weight ?? undefined,
        width: product?.width ?? undefined,
        height: product?.height ?? undefined,
        depth: product?.depth ?? undefined,
        image_url: product?.image_url ?? "",
      }}
      onSubmit={handleFormSubmit}
      className="space-y-4"
      submitLabel={product ? t("form.submitUpdate") : t("form.submitCreate")}
      submittingLabel={tc("saving")}
      isSubmitting={isLoading}
      showErrorSummary={false}
    >
      {({ register, setValue, watch, formState: { errors } }) => {
        const [sourceValue, imageUrlValue, descShortValue, descLongValue] = watch([
          "source",
          "image_url",
          "description_short",
          "description_long",
        ]);

        return (
          <>
            <ProductBasicFields
              register={register}
              setValue={setValue}
              errors={errors}
              sourceValue={sourceValue}
              selectedCategoryId={selectedCategoryId}
              onCategoryChange={setSelectedCategoryId}
            />

            <ProductDescriptionFields
              register={register}
              setValue={setValue}
              descShortValue={descShortValue}
              descLongValue={descLongValue}
            />

            <ProductDimensionsFields register={register} errors={errors} />

            <ProductImagesFields
              register={register}
              imageUrlValue={imageUrlValue}
              imageList={imageList}
              onMainImageUploaded={(url) => setValue("image_url", url, { shouldValidate: true })}
              onAddImage={addImage}
              onRemoveImage={removeImage}
              onUpdateImage={updateImage}
              onAppendUploadedImage={(url) => setImageList((prev) => [...prev, { url, alt: "" }])}
            />

            <Separator />
            <ProductDropshipFields
              isDropship={isDropship}
              dropshipSupplierId={dropshipSupplierId}
              onDropshipChange={setIsDropship}
              onSupplierChange={setDropshipSupplierId}
            />

            <Separator />
            <div className="space-y-2">
              <Label>{tc("tags")}</Label>
              <TagInput tags={tags} onChange={setTags} />
            </div>
          </>
        );
      }}
    </FormWrapper>
  );
}
