"use client";

import { useState, useRef } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus, Trash2, ImageIcon, Upload, Loader2 } from "lucide-react";
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
import { TagInput } from "@/components/shared/tag-input";
import type { Product, CreateProductRequest, UpdateProductRequest } from "@/types/api";
import { uploadFile } from "@/lib/api-client";
import { toast } from "sonner";
import { useProductCategories } from "@/hooks/use-product-categories";

const PRODUCT_SOURCES = ["manual", "allegro", "woocommerce"] as const;

const PRODUCT_SOURCE_LABELS: Record<string, string> = {
  manual: "Ręczne",
  allegro: "Allegro",
  woocommerce: "WooCommerce",
};

const productSchema = z.object({
  name: z.string().min(1, "Nazwa produktu jest wymagana"),
  sku: z.string().optional(),
  ean: z.string().optional(),
  price: z.number().min(0, "Cena musi być większa lub równa 0"),
  stock_quantity: z
    .number()
    .int("Ilość musi być liczbą całkowitą")
    .min(0, "Ilość musi być większa lub równa 0"),
  source: z.enum(["manual", "allegro", "woocommerce"]),
  description_short: z.string().optional(),
  description_long: z.string().optional(),
  weight: z.number().min(0).optional(),
  width: z.number().min(0).optional(),
  height: z.number().min(0).optional(),
  depth: z.number().min(0).optional(),
  image_url: z.string(),
});

type ProductFormValues = z.infer<typeof productSchema>;

interface ProductFormProps {
  product?: Product;
  onSubmit: (data: CreateProductRequest) => void;
  isLoading?: boolean;
}

export function ProductForm({ product, onSubmit, isLoading }: ProductFormProps) {
  const [imageList, setImageList] = useState<{ url: string; alt: string }[]>(
    product?.images?.map((img) => ({ url: img.url, alt: img.alt || "" })) ?? []
  );
  const [tags, setTags] = useState<string[]>(product?.tags || []);
  const [selectedCategory, setSelectedCategory] = useState<string>(product?.category || "");
  const [uploadingMain, setUploadingMain] = useState(false);
  const [uploadingIdx, setUploadingIdx] = useState<number | null>(null);
  const mainFileRef = useRef<HTMLInputElement>(null);
  const galleryFileRef = useRef<HTMLInputElement>(null);
  const { data: categoriesConfig } = useProductCategories();

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
      category: selectedCategory && selectedCategory !== "__none__" ? selectedCategory : undefined,
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
        <Label htmlFor="name">Nazwa produktu</Label>
        <Input
          id="name"
          placeholder="Nazwa produktu"
          {...register("name")}
        />
        {errors.name && (
          <p className="text-sm text-destructive">{errors.name.message}</p>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="sku">SKU</Label>
          <Input
            id="sku"
            placeholder="Opcjonalny kod SKU"
            {...register("sku")}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="ean">EAN</Label>
          <Input
            id="ean"
            placeholder="Opcjonalny kod EAN"
            {...register("ean")}
          />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="price">Cena</Label>
          <Input
            id="price"
            type="number"
            step="0.01"
            min="0"
            placeholder="0.00"
            {...register("price", { valueAsNumber: true })}
          />
          {errors.price && (
            <p className="text-sm text-destructive">{errors.price.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="stock_quantity">Stan magazynowy</Label>
          <Input
            id="stock_quantity"
            type="number"
            step="1"
            min="0"
            placeholder="0"
            {...register("stock_quantity", { valueAsNumber: true })}
          />
          {errors.stock_quantity && (
            <p className="text-sm text-destructive">
              {errors.stock_quantity.message}
            </p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="source">Źródło</Label>
        <Select
          value={sourceValue}
          onValueChange={(value) =>
            setValue("source", value as ProductFormValues["source"], {
              shouldValidate: true,
            })
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Wybierz źródło" />
          </SelectTrigger>
          <SelectContent>
            {PRODUCT_SOURCES.map((source) => (
              <SelectItem key={source} value={source}>
                {PRODUCT_SOURCE_LABELS[source]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {errors.source && (
          <p className="text-sm text-destructive">{errors.source.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="category">Kategoria</Label>
        <Select value={selectedCategory} onValueChange={setSelectedCategory}>
          <SelectTrigger>
            <SelectValue placeholder="Wybierz kategorię" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">Brak kategorii</SelectItem>
            {categoriesConfig?.categories
              ?.sort((a, b) => a.position - b.position)
              .map((cat) => (
                <SelectItem key={cat.key} value={cat.key}>
                  {cat.label}
                </SelectItem>
              ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label htmlFor="description_short">Krótki opis</Label>
        <Input
          id="description_short"
          placeholder="Krótki opis produktu..."
          {...register("description_short")}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="description_long">Opis</Label>
        <Textarea
          id="description_long"
          placeholder="Pełny opis produktu..."
          rows={5}
          {...register("description_long")}
        />
      </div>

      <div className="space-y-4">
        <h3 className="text-sm font-medium">Wymiary i waga</h3>
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
          <div className="space-y-2">
            <Label htmlFor="weight">Waga (kg)</Label>
            <Input id="weight" type="number" step="0.001" min="0" placeholder="0.000" {...register("weight", { valueAsNumber: true })} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="width">Szerokość (cm)</Label>
            <Input id="width" type="number" step="0.01" min="0" placeholder="0.00" {...register("width", { valueAsNumber: true })} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="height">Wysokość (cm)</Label>
            <Input id="height" type="number" step="0.01" min="0" placeholder="0.00" {...register("height", { valueAsNumber: true })} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="depth">Głębokość (cm)</Label>
            <Input id="depth" type="number" step="0.01" min="0" placeholder="0.00" {...register("depth", { valueAsNumber: true })} />
          </div>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="image_url">Zdjęcie główne (URL)</Label>
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
            Wgraj
          </Button>
          <input
            ref={mainFileRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            className="hidden"
            onChange={async (e) => {
              const file = e.target.files?.[0];
              if (!file) return;
              setUploadingMain(true);
              try {
                const { url } = await uploadFile(file);
                setValue("image_url", url, { shouldValidate: true });
              } catch (err) {
                toast.error(err instanceof Error ? err.message : "Błąd uploadu");
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
            alt="Podgląd zdjęcia głównego"
            className="h-32 w-32 rounded-lg object-cover border"
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = "none";
            }}
          />
        )}
      </div>

      <div className="space-y-2">
        <Label>Dodatkowe zdjęcia</Label>
        <div className="space-y-2">
          {imageList.map((img, index) => (
            <div key={index} className="flex items-start gap-2">
              <div className="flex-1 space-y-1">
                <Input
                  placeholder="URL zdjęcia"
                  value={img.url}
                  onChange={(e) => updateImage(index, "url", e.target.value)}
                />
                <Input
                  placeholder="Tekst alternatywny (opcjonalny)"
                  value={img.alt}
                  onChange={(e) => updateImage(index, "alt", e.target.value)}
                />
              </div>
              {img.url.trim() !== "" && (
                <img
                  src={img.url}
                  alt={img.alt || `Zdjęcie ${index + 1}`}
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
              Dodaj zdjęcie
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
            Wgraj zdjęcie
          </Button>
          <input
            ref={galleryFileRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            className="hidden"
            onChange={async (e) => {
              const file = e.target.files?.[0];
              if (!file) return;
              setUploadingIdx(imageList.length);
              try {
                const { url } = await uploadFile(file);
                setImageList((prev) => [...prev, { url, alt: "" }]);
              } catch (err) {
                toast.error(err instanceof Error ? err.message : "Błąd uploadu");
              } finally {
                setUploadingIdx(null);
                e.target.value = "";
              }
            }}
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label>Tagi</Label>
        <TagInput tags={tags} onChange={setTags} />
      </div>

      <Button type="submit" disabled={isLoading}>
        {isLoading
          ? "Zapisywanie..."
          : product
            ? "Zapisz zmiany"
            : "Utwórz produkt"}
      </Button>
    </form>
  );
}
