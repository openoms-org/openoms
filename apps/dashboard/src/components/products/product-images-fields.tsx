"use client";

import { useRef, useState } from "react";
import { Loader2, Plus, Trash2, Upload } from "lucide-react";
import { useTranslations } from "next-intl";
import type { UseFormRegister } from "react-hook-form";
import { toast } from "sonner";
import { FormField } from "@/components/shared/form-wrapper";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { uploadFile } from "@/lib/api-client";
import type { ProductFormValues } from "./product-form.schema";

interface ProductImageItem {
  url: string;
  alt: string;
}

interface ProductImagesFieldsProps {
  register: UseFormRegister<ProductFormValues>;
  imageUrlValue: string | undefined;
  imageList: ProductImageItem[];
  onMainImageUploaded: (url: string) => void;
  onAddImage: () => void;
  onRemoveImage: (index: number) => void;
  onUpdateImage: (index: number, field: "url" | "alt", value: string) => void;
  onAppendUploadedImage: (url: string) => void;
}

export function ProductImagesFields({
  register,
  imageUrlValue,
  imageList,
  onMainImageUploaded,
  onAddImage,
  onRemoveImage,
  onUpdateImage,
  onAppendUploadedImage,
}: ProductImagesFieldsProps) {
  const t = useTranslations("products");
  const [uploadingMain, setUploadingMain] = useState(false);
  const [uploadingIdx, setUploadingIdx] = useState<number | null>(null);
  const [brokenImages, setBrokenImages] = useState<Record<string, true>>({});
  const mainFileRef = useRef<HTMLInputElement>(null);
  const galleryFileRef = useRef<HTMLInputElement>(null);
  const markImageBroken = (key: string) => {
    setBrokenImages((current) => ({ ...current, [key]: true }));
  };

  return (
    <>
      <FormField<ProductFormValues> name="image_url" label={t("form.mainImage")}>
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
            {uploadingMain ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Upload className="h-4 w-4" />
            )}
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
                onMainImageUploaded(url);
              } catch (err) {
                toast.error(err instanceof Error ? err.message : t("form.uploadError"));
              } finally {
                setUploadingMain(false);
                e.target.value = "";
              }
            }}
          />
        </div>
        {imageUrlValue && imageUrlValue.trim() !== "" && !brokenImages[`main:${imageUrlValue}`] && (
          <img
            src={imageUrlValue}
            alt={t("form.mainImagePreview")}
            className="h-32 w-32 rounded-lg object-cover border"
            onError={() => markImageBroken(`main:${imageUrlValue}`)}
          />
        )}
      </FormField>

      <FormField<ProductFormValues> label={t("form.additionalImages")}>
        <div className="space-y-2">
          {imageList.map((img, index) => (
            <div key={index} className="flex items-start gap-2">
              <div className="flex-1 space-y-1">
                <Input
                  placeholder={t("form.imageUrl")}
                  value={img.url}
                  onChange={(e) => onUpdateImage(index, "url", e.target.value)}
                />
                <Input
                  placeholder={t("form.imageAlt")}
                  value={img.alt}
                  onChange={(e) => onUpdateImage(index, "alt", e.target.value)}
                />
              </div>
              {img.url.trim() !== "" && !brokenImages[`gallery:${index}:${img.url}`] && (
                <img
                  src={img.url}
                  alt={img.alt || `${t("form.mainImagePreview")} ${index + 1}`}
                  className="h-10 w-10 rounded border object-cover"
                  onError={() => markImageBroken(`gallery:${index}:${img.url}`)}
                />
              )}
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => onRemoveImage(index)}
              >
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </div>
          ))}
        </div>
        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={imageList.length >= 16}
            onClick={onAddImage}
          >
            <Plus className="h-4 w-4" />
            {t("form.addImage")}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={uploadingIdx !== null || imageList.length >= 16}
            onClick={() => galleryFileRef.current?.click()}
          >
            {uploadingIdx !== null ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Upload className="h-4 w-4" />
            )}
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
                onAppendUploadedImage(url);
              } catch (err) {
                toast.error(err instanceof Error ? err.message : t("form.uploadError"));
              } finally {
                setUploadingIdx(null);
                e.target.value = "";
              }
            }}
          />
        </div>
      </FormField>
    </>
  );
}
