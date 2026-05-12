"use client";

import { useState } from "react";
import { ChevronDown, Languages, Loader2, Sparkles } from "lucide-react";
import { useTranslations } from "next-intl";
import type { UseFormRegister, UseFormSetValue } from "react-hook-form";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useImproveDescription, useTranslateDescription } from "@/hooks/use-ai";
import { getErrorMessage } from "@/lib/api-client";
import type { ProductFormValues } from "./product-form.schema";

type DescriptionField = "description_short" | "description_long";

interface ProductDescriptionFieldsProps {
  register: UseFormRegister<ProductFormValues>;
  setValue: UseFormSetValue<ProductFormValues>;
  descShortValue: string | undefined;
  descLongValue: string | undefined;
}

export function ProductDescriptionFields({
  register,
  setValue,
  descShortValue,
  descLongValue,
}: ProductDescriptionFieldsProps) {
  const t = useTranslations("products");
  const improveDescription = useImproveDescription();
  const translateDescription = useTranslateDescription();
  const [pendingImproveField, setPendingImproveField] = useState<DescriptionField | null>(null);
  const [pendingTranslateField, setPendingTranslateField] = useState<DescriptionField | null>(null);

  const handleImprove = (field: DescriptionField) => {
    const text = field === "description_short" ? descShortValue : descLongValue;
    if (!text?.trim()) {
      toast.error(t("form.descriptionEmpty"));
      return;
    }
    setPendingImproveField(field);
    improveDescription.mutate(
      { description: text },
      {
        onSuccess: (data) => {
          setValue(field, data.description, { shouldDirty: true });
          toast.success(t("form.improvedByAi"));
        },
        onError: (error) => toast.error(getErrorMessage(error)),
        onSettled: () => setPendingImproveField(null),
      }
    );
  };

  const handleTranslate = (
    field: DescriptionField,
    targetLanguage: string
  ) => {
    const text = field === "description_short" ? descShortValue : descLongValue;
    if (!text?.trim()) {
      toast.error(t("form.descriptionEmpty"));
      return;
    }
    setPendingTranslateField(field);
    translateDescription.mutate(
      { description: text, target_language: targetLanguage },
      {
        onSuccess: (data) => {
          setValue(field, data.description, { shouldDirty: true });
          toast.success(t("form.translatedByAi"));
        },
        onError: (error) => toast.error(getErrorMessage(error)),
        onSettled: () => setPendingTranslateField(null),
      }
    );
  };

  return (
    <>
      <div className="space-y-2">
        <FieldHeader
          htmlFor="description_short"
          label={t("form.shortDescription")}
          disabled={!descShortValue?.trim()}
          isImproving={pendingImproveField === "description_short"}
          isTranslating={pendingTranslateField === "description_short"}
          onImprove={() => handleImprove("description_short")}
          onTranslate={(language) => handleTranslate("description_short", language)}
        />
        <Input
          id="description_short"
          placeholder={t("form.shortDescriptionPlaceholder")}
          {...register("description_short")}
        />
      </div>

      <div className="space-y-2">
        <FieldHeader
          htmlFor="description_long"
          label={t("form.longDescription")}
          disabled={!descLongValue?.trim()}
          isImproving={pendingImproveField === "description_long"}
          isTranslating={pendingTranslateField === "description_long"}
          onImprove={() => handleImprove("description_long")}
          onTranslate={(language) => handleTranslate("description_long", language)}
        />
        <Textarea
          id="description_long"
          placeholder={t("form.longDescriptionPlaceholder")}
          rows={5}
          {...register("description_long")}
        />
      </div>
    </>
  );
}

interface FieldHeaderProps {
  htmlFor: string;
  label: string;
  disabled: boolean;
  isImproving: boolean;
  isTranslating: boolean;
  onImprove: () => void;
  onTranslate: (language: string) => void;
}

function FieldHeader({
  htmlFor,
  label,
  disabled,
  isImproving,
  isTranslating,
  onImprove,
  onTranslate,
}: FieldHeaderProps) {
  const t = useTranslations("products");

  return (
    <div className="flex w-full items-center justify-between">
      <Label htmlFor={htmlFor}>{label}</Label>
      <div className="flex items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs"
          disabled={disabled || isImproving}
          onClick={onImprove}
        >
          {isImproving ? (
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
              disabled={disabled || isTranslating}
            >
              {isTranslating ? (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              ) : (
                <Languages className="mr-1 h-3 w-3" />
              )}
              {t("form.translate")}
              <ChevronDown className="ml-1 h-3 w-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => onTranslate("pl")}>
              {t("form.languages.polish")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onTranslate("en")}>
              {t("form.languages.english")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onTranslate("de")}>
              {t("form.languages.german")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
