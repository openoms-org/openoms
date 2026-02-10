"use client";

import { useState, useRef } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus, Trash2, ImageIcon, Upload, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Product } from "@/types/api";
import { uploadFile } from "@/lib/api-client";
import { toast } from "sonner";

const PRODUCT_SOURCES = ["manual", "allegro", "woocommerce"] as const;

const PRODUCT_SOURCE_LABELS: Record<string, string> = {
  manual: "Reczne",
  allegro: "Allegro",
  woocommerce: "WooCommerce",
};

const productSchema = z.object({
  name: z.string().min(1, "Nazwa produktu jest wymagana"),
  sku: z.string().optional(),
  ean: z.string().optional(),
  price: z.number().min(0, "Cena musi byc wieksza lub rowna 0"),
  stock_quantity: z
    .number()
    .int("Ilosc musi byc liczba calkowita")
    .min(0, "Ilosc musi byc wieksza lub rowna 0"),
  source: z.enum(["manual", "allegro", "woocommerce"]),
  image_url: z.string(),
});

type ProductFormValues = z.infer<typeof productSchema>;

interface ProductFormProps {
  product?: Product;
  onSubmit: (data: any) => void;
  isLoading?: boolean;
}

export function ProductForm({ product, onSubmit, isLoading }: ProductFormProps) {
  const [imageList, setImageList] = useState<{ url: string; alt: string }[]>(
    product?.images?.map((img) => ({ url: img.url, alt: img.alt || "" })) ?? []
  );
  const [uploadingMain, setUploadingMain] = useState(false);
  const [uploadingIdx, setUploadingIdx] = useState<number | null>(null);
  const mainFileRef = useRef<HTMLInputElement>(null);
  const galleryFileRef = useRef<HTMLInputElement>(null);

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
      image_url: product?.image_url ?? "",
    },
  });

  const sourceValue = watch("source");
  const imageUrlValue = watch("image_url");

  const handleFormSubmit = (data: ProductFormValues) => {
    onSubmit({
      ...data,
      image_url: data.image_url || undefined,
      images: imageList
        .filter((img) => img.url.trim() !== "")
        .map((img, i) => ({ url: img.url, alt: img.alt || undefined, position: i + 1 })),
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
        <Label htmlFor="source">Zrodlo</Label>
        <Select
          value={sourceValue}
          onValueChange={(value) =>
            setValue("source", value as ProductFormValues["source"], {
              shouldValidate: true,
            })
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Wybierz zrodlo" />
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
        <Label htmlFor="image_url">Zdjecie glowne (URL)</Label>
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
                toast.error(err instanceof Error ? err.message : "Blad uploadu");
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
            alt="Podglad zdjecia glownego"
            className="h-32 w-32 rounded-lg object-cover border"
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = "none";
            }}
          />
        )}
      </div>

      <div className="space-y-2">
        <Label>Dodatkowe zdjecia</Label>
        <div className="space-y-2">
          {imageList.map((img, index) => (
            <div key={index} className="flex items-start gap-2">
              <div className="flex-1 space-y-1">
                <Input
                  placeholder="URL zdjecia"
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
                  alt={img.alt || `Zdjecie ${index + 1}`}
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
              Dodaj zdjecie
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
            Wgraj zdjecie
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
                toast.error(err instanceof Error ? err.message : "Blad uploadu");
              } finally {
                setUploadingIdx(null);
                e.target.value = "";
              }
            }}
          />
        </div>
      </div>

      <Button type="submit" disabled={isLoading}>
        {isLoading
          ? "Zapisywanie..."
          : product
            ? "Zapisz zmiany"
            : "Utworz produkt"}
      </Button>
    </form>
  );
}
