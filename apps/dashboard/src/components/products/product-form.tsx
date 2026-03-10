"use client";

import { useState, useRef } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus, Trash2, Upload, Loader2, Sparkles, Languages, ChevronDown } from "lucide-react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { TagInput } from "@/components/shared/tag-input";
import { Separator } from "@/components/ui/separator";
import type { Product, CreateProductRequest, UpdateProductRequest } from "@/types/api";
import { normalizeProductImages } from "@/types/api";
import { uploadFile, getErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";
import { CategoryTreePicker } from "@/components/shared/category-tree-picker";
import { useImproveDescription, useTranslateDescription } from "@/hooks/use-ai";
import { useSuppliers } from "@/hooks/use-suppliers";
import { Switch } from "@/components/ui/switch";

const PRODUCT_SOURCES = ["manual", "allegro", "woocommerce"] as const;

// Empty number inputs with valueAsNumber produce NaN — catch converts to undefined
const optionalDimension = z.number().min(0).optional().catch(undefined);

function createProductSchema(t: (key: string) => string) {
  return z.object({
    name: z.string().min(1, t("nameRequired")),
    sku: z.string().optional(),
    ean: z.string().optional(),
    price: z.number().min(0, t("priceMin")),
    stock_quantity: z
      .number()
      .int(t("quantityInteger"))
      .min(0, t("quantityMin")),
    source: z.enum(["manual", "allegro", "woocommerce"]),
    description_short: z.string().optional(),
    description_long: z.string().optional(),
    weight: optionalDimension,
    width: optionalDimension,
    height: optionalDimension,
    depth: optionalDimension,
    image_url: z.string(),
  });
}

type ProductFormValues = z.infer<ReturnType<typeof createProductSchema>>;

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
  const [selectedCategoryId, setSelectedCategoryId] = useState<string | undefined>(product?.category_id || undefined);
  const [isDropship, setIsDropship] = useState(product?.is_dropship || false);
  const [dropshipSupplierId, setDropshipSupplierId] = useState<string>(product?.dropship_supplier_id || "");
  const { data: suppliersData } = useSuppliers({ limit: 100 });
  const [uploadingMain, setUploadingMain] = useState(false);
  const [uploadingIdx, setUploadingIdx] = useState<number | null>(null);
  const mainFileRef = useRef<HTMLInputElement>(null);
  const galleryFileRef = useRef<HTMLInputElement>(null);
  const improveDescription = useImproveDescription();
  const translateDescription = useTranslateDescription();

  const productSchema = createProductSchema(tv);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<ProductFormValues>({
    resolver: zodResolver(productSchema),
    defaultValues: {
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
    },
  });

  const sourceValue = watch("source");
  const imageUrlValue = watch("image_url");
  const descShortValue = watch("description_short");
  const descLongValue = watch("description_long");

  const handleImprove = (field: "description_short" | "description_long") => {
    const text = field === "description_short" ? descShortValue : descLongValue;
    if (!text?.trim()) {
      toast.error(t("form.descriptionEmpty"));
      return;
    }
    improveDescription.mutate(
      { description: text },
      {
        onSuccess: (data) => {
          setValue(field, data.description, { shouldDirty: true });
          toast.success(t("form.improvedByAi"));
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  const handleTranslate = (field: "description_short" | "description_long", targetLanguage: string) => {
    const text = field === "description_short" ? descShortValue : descLongValue;
    if (!text?.trim()) {
      toast.error(t("form.descriptionEmpty"));
      return;
    }
    translateDescription.mutate(
      { description: text, target_language: targetLanguage },
      {
        onSuccess: (data) => {
          setValue(field, data.description, { shouldDirty: true });
          toast.success(t("form.translatedByAi"));
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

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
    if (imageList.length >= 16) return;
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
    <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="name">{t("form.productName")} <span className="text-destructive">*</span></Label>
        <Input
          id="name"
          placeholder={t("form.productNamePlaceholder")}
          aria-invalid={!!errors.name}
          {...register("name")}
        />
        {errors.name && (
          <p className="text-destructive text-xs mt-1">{errors.name.message}</p>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="sku">{t("form.sku")}</Label>
          <Input
            id="sku"
            placeholder={t("form.skuPlaceholder")}
            {...register("sku")}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="ean">{t("form.ean")}</Label>
          <Input
            id="ean"
            placeholder={t("form.eanPlaceholder")}
            {...register("ean")}
          />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="price">{t("form.price")} <span className="text-destructive">*</span></Label>
          <Input
            id="price"
            type="number"
            step="0.01"
            min="0"
            placeholder="0.00"
            aria-invalid={!!errors.price}
            {...register("price", { valueAsNumber: true })}
          />
          {errors.price && (
            <p className="text-destructive text-xs mt-1">{errors.price.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="stock_quantity">{t("form.stockQuantity")} <span className="text-destructive">*</span></Label>
          <Input
            id="stock_quantity"
            type="number"
            step="1"
            min="0"
            placeholder="0"
            aria-invalid={!!errors.stock_quantity}
            {...register("stock_quantity", { valueAsNumber: true })}
          />
          {errors.stock_quantity && (
            <p className="text-destructive text-xs mt-1">
              {errors.stock_quantity.message}
            </p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="source">{t("form.source")} <span className="text-destructive">*</span></Label>
        <Select
          value={sourceValue}
          onValueChange={(value) =>
            setValue("source", value as ProductFormValues["source"], {
              shouldValidate: true,
            })
          }
        >
          <SelectTrigger className="w-full" aria-invalid={!!errors.source}>
            <SelectValue placeholder={t("form.sourcePlaceholder")} />
          </SelectTrigger>
          <SelectContent>
            {PRODUCT_SOURCES.map((source) => (
              <SelectItem key={source} value={source}>
                {source === "manual" ? t("form.sourceManual") : source === "allegro" ? "Allegro" : "WooCommerce"}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {errors.source && (
          <p className="text-destructive text-xs mt-1">{errors.source.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label>{t("form.category")}</Label>
        <CategoryTreePicker
          value={selectedCategoryId}
          onChange={setSelectedCategoryId}
          placeholder={t("form.categoryPlaceholder")}
          className="w-full"
        />
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label htmlFor="description_short">{t("form.shortDescription")}</Label>
          <div className="flex items-center gap-1">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs"
              disabled={!descShortValue?.trim() || improveDescription.isPending}
              onClick={() => handleImprove("description_short")}
            >
              {improveDescription.isPending ? (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              ) : (
                <Sparkles className="mr-1 h-3 w-3" />
              )}
              {t("form.improve")}
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 text-xs"
                  disabled={!descShortValue?.trim() || translateDescription.isPending}
                >
                  {translateDescription.isPending ? (
                    <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                  ) : (
                    <Languages className="mr-1 h-3 w-3" />
                  )}
                  {t("form.translate")}
                  <ChevronDown className="ml-1 h-3 w-3" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleTranslate("description_short", "pl")}>
                  Polski
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleTranslate("description_short", "en")}>
                  English
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleTranslate("description_short", "de")}>
                  Deutsch
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        <Input
          id="description_short"
          placeholder={t("form.shortDescriptionPlaceholder")}
          {...register("description_short")}
        />
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label htmlFor="description_long">{t("form.longDescription")}</Label>
          <div className="flex items-center gap-1">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs"
              disabled={!descLongValue?.trim() || improveDescription.isPending}
              onClick={() => handleImprove("description_long")}
            >
              {improveDescription.isPending ? (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              ) : (
                <Sparkles className="mr-1 h-3 w-3" />
              )}
              {t("form.improve")}
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 text-xs"
                  disabled={!descLongValue?.trim() || translateDescription.isPending}
                >
                  {translateDescription.isPending ? (
                    <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                  ) : (
                    <Languages className="mr-1 h-3 w-3" />
                  )}
                  {t("form.translate")}
                  <ChevronDown className="ml-1 h-3 w-3" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleTranslate("description_long", "pl")}>
                  Polski
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleTranslate("description_long", "en")}>
                  English
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleTranslate("description_long", "de")}>
                  Deutsch
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        <Textarea
          id="description_long"
          placeholder={t("form.longDescriptionPlaceholder")}
          rows={5}
          {...register("description_long")}
        />
      </div>

      <div className="space-y-4">
        <h3 className="text-sm font-medium">{t("form.dimensionsAndWeight")}</h3>
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
          <div className="space-y-2">
            <Label htmlFor="weight">{t("form.weight")}</Label>
            <Input id="weight" type="number" step="0.001" min="0" placeholder="0.000" {...register("weight", { valueAsNumber: true })} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="width">{t("form.width")}</Label>
            <Input id="width" type="number" step="0.01" min="0" placeholder="0.00" {...register("width", { valueAsNumber: true })} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="height">{t("form.height")}</Label>
            <Input id="height" type="number" step="0.01" min="0" placeholder="0.00" {...register("height", { valueAsNumber: true })} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="depth">{t("form.depth")}</Label>
            <Input id="depth" type="number" step="0.01" min="0" placeholder="0.00" {...register("depth", { valueAsNumber: true })} />
          </div>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="image_url">{t("form.mainImage")}</Label>
        <div className="flex gap-2">
          <Input
            id="image_url"
            className="flex-1"
            placeholder="https://example.com/image.jpg"
            {...register("image_url")}
          />
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={uploadingMain}
            onClick={() => mainFileRef.current?.click()}
          >
            {uploadingMain ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
            {t("form.uploadImage")}
          </Button>
          <input
            ref={mainFileRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            className="hidden"
            onChange={async (e) => {
              const file = e.target.files?.[0];
              if (!file) return;
              if (file.size > 5 * 1024 * 1024) {
                toast.error(t("form.fileTooLarge"));
                e.target.value = "";
                return;
              }
              setUploadingMain(true);
              try {
                const { url } = await uploadFile(file);
                setValue("image_url", url, { shouldValidate: true });
              } catch (err) {
                toast.error(err instanceof Error ? err.message : t("form.uploadError"));
              } finally {
                setUploadingMain(false);
                e.target.value = "";
              }
            }}
          />
        </div>
        {imageUrlValue && imageUrlValue.trim() !== "" && (
          <img
            src={imageUrlValue}
            alt={t("form.mainImagePreview")}
            className="h-32 w-32 rounded-lg object-cover border"
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = "none";
            }}
          />
        )}
      </div>

      <div className="space-y-2">
        <Label>{t("form.additionalImages")}</Label>
        <div className="space-y-2">
          {imageList.map((img, index) => (
            <div key={index} className="flex items-start gap-2">
              <div className="flex-1 space-y-1">
                <Input
                  placeholder={t("form.imageUrl")}
                  value={img.url}
                  onChange={(e) => updateImage(index, "url", e.target.value)}
                />
                <Input
                  placeholder={t("form.imageAlt")}
                  value={img.alt}
                  onChange={(e) => updateImage(index, "alt", e.target.value)}
                />
              </div>
              {img.url.trim() !== "" && (
                <img
                  src={img.url}
                  alt={img.alt || `${t("form.mainImagePreview")} ${index + 1}`}
                  className="h-10 w-10 rounded border object-cover"
                  onError={(e) => {
                    (e.target as HTMLImageElement).style.display = "none";
                  }}
                />
              )}
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => removeImage(index)}
              >
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </div>
          ))}
        </div>
        <div className="flex gap-2">
          {imageList.length < 16 && (
            <Button type="button" variant="outline" size="sm" onClick={addImage}>
              <Plus className="h-4 w-4" />
              {t("form.addImage")}
            </Button>
          )}
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={uploadingIdx !== null || imageList.length >= 16}
            onClick={() => galleryFileRef.current?.click()}
          >
            {uploadingIdx !== null ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
            {t("form.uploadPhoto")}
          </Button>
          <input
            ref={galleryFileRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            className="hidden"
            onChange={async (e) => {
              const file = e.target.files?.[0];
              if (!file) return;
              if (file.size > 5 * 1024 * 1024) {
                toast.error(t("form.fileTooLarge"));
                e.target.value = "";
                return;
              }
              setUploadingIdx(imageList.length);
              try {
                const { url } = await uploadFile(file);
                setImageList((prev) => [...prev, { url, alt: "" }]);
              } catch (err) {
                toast.error(err instanceof Error ? err.message : t("form.uploadError"));
              } finally {
                setUploadingIdx(null);
                e.target.value = "";
              }
            }}
          />
        </div>
      </div>

      <Separator />

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label>{t("form.dropshipProduct")}</Label>
            <p className="text-xs text-muted-foreground">
              {t("form.dropshipDescription")}
            </p>
          </div>
          <Switch checked={isDropship} onCheckedChange={setIsDropship} />
        </div>
        {isDropship && (
          <div className="space-y-2">
            <Label>{t("form.dropshipSupplier")}</Label>
            <Select value={dropshipSupplierId} onValueChange={setDropshipSupplierId}>
              <SelectTrigger>
                <SelectValue placeholder={t("form.dropshipSupplierPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                {suppliersData?.items?.map((s) => (
                  <SelectItem key={s.id} value={s.id}>
                    {s.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
      </div>

      <Separator />

      <div className="space-y-2">
        <Label>{tc("tags")}</Label>
        <TagInput tags={tags} onChange={setTags} />
      </div>

      <Button type="submit" disabled={isLoading}>
        {isLoading
          ? tc("saving")
          : product
            ? t("form.submitUpdate")
            : t("form.submitCreate")}
      </Button>
    </form>
  );
}
