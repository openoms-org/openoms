"use client";

import { useTranslations } from "next-intl";
import type { FieldErrors, UseFormRegister } from "react-hook-form";
import { TagInput } from "@/components/shared/tag-input";
import { FormField } from "@/components/shared/form-wrapper";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { ORDER_PRIORITIES } from "@/lib/constants";
import type { CustomFieldDef } from "@/types/api";
import type { OrderFormValues, OrderPriority } from "./order-form.schema";

interface OrderMetadataFieldsProps {
  register: UseFormRegister<OrderFormValues>;
  errors: FieldErrors<OrderFormValues>;
  customFields: CustomFieldDef[];
  customValues: Record<string, unknown>;
  tags: string[];
  priority: OrderPriority;
  internalNotes: string;
  onCustomFieldChange: (key: string, value: unknown) => void;
  onTagsChange: (tags: string[]) => void;
  onPriorityChange: (priority: OrderPriority) => void;
  onInternalNotesChange: (notes: string) => void;
}

export function OrderMetadataFields({
  register,
  errors,
  customFields,
  customValues,
  tags,
  priority,
  internalNotes,
  onCustomFieldChange,
  onTagsChange,
  onPriorityChange,
  onInternalNotesChange,
}: OrderMetadataFieldsProps) {
  const t = useTranslations("orders");
  const tc = useTranslations("common");

  return (
    <>
      <FormField<OrderFormValues>
        name="notes"
        label={tc("notes")}
        error={errors.notes}
      >
        <Textarea
          id="notes"
          placeholder={t("form.notesPlaceholder")}
          rows={3}
          {...register("notes")}
        />
      </FormField>

      <FormField<OrderFormValues> label={t("form.priority")} htmlFor="priority">
        <Select value={priority} onValueChange={(value) => onPriorityChange(value as OrderPriority)}>
          <SelectTrigger id="priority">
            <SelectValue placeholder={t("form.priorityPlaceholder")} />
          </SelectTrigger>
          <SelectContent>
            {Object.entries(ORDER_PRIORITIES).map(([key, { label }]) => (
              <SelectItem key={key} value={key}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FormField>

      <FormField<OrderFormValues> label={t("form.internalNotes")} htmlFor="internal_notes">
        <Textarea
          id="internal_notes"
          placeholder={t("form.internalNotesPlaceholder")}
          rows={3}
          value={internalNotes}
          onChange={(e) => onInternalNotesChange(e.target.value)}
          className="border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-950/30"
        />
      </FormField>

      {customFields.length > 0 && (
        <>
          <Separator />
          <div>
            <h3 className="text-sm font-medium mb-4">{t("form.customFields")}</h3>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            {[...customFields]
              .sort((a, b) => a.position - b.position)
              .map((field) => (
                <FormField<OrderFormValues>
                  key={field.key}
                  label={
                    <>
                      {field.label}
                      {field.required && <span className="text-destructive ml-1">*</span>}
                    </>
                  }
                  htmlFor={`cf_${field.key}`}
                >
                  {renderCustomField(field, customValues, onCustomFieldChange, tc("selectPlaceholder"))}
                </FormField>
              ))}
            </div>
          </div>
        </>
      )}

      <div className="space-y-2">
        <Label>{tc("tags")}</Label>
        <TagInput tags={tags} onChange={onTagsChange} />
      </div>
    </>
  );
}

function renderCustomField(
  field: CustomFieldDef,
  customValues: Record<string, unknown>,
  onCustomFieldChange: (key: string, value: unknown) => void,
  selectPlaceholder: string
) {
  switch (field.type) {
    case "text":
      return (
        <Input
          id={`cf_${field.key}`}
          value={(customValues[field.key] as string) || ""}
          onChange={(e) => onCustomFieldChange(field.key, e.target.value)}
        />
      );
    case "number":
      return (
        <Input
          id={`cf_${field.key}`}
          type="number"
          step="any"
          value={(customValues[field.key] as string | number) ?? ""}
          onChange={(e) =>
            onCustomFieldChange(
              field.key,
              e.target.value === "" ? "" : Number(e.target.value)
            )
          }
        />
      );
    case "select":
      return (
        <Select
          value={(customValues[field.key] as string) || ""}
          onValueChange={(value) => onCustomFieldChange(field.key, value)}
        >
          <SelectTrigger>
            <SelectValue placeholder={selectPlaceholder} />
          </SelectTrigger>
          <SelectContent>
            {(field.options || []).map((option) => (
              <SelectItem key={option} value={option}>
                {option}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    case "date":
      return (
        <Input
          id={`cf_${field.key}`}
          type="date"
          value={(customValues[field.key] as string) || ""}
          onChange={(e) => onCustomFieldChange(field.key, e.target.value)}
        />
      );
    case "checkbox":
      return (
        <div className="flex items-center gap-2 pt-1">
          <input
            id={`cf_${field.key}`}
            type="checkbox"
            checked={!!customValues[field.key]}
            onChange={(e) => onCustomFieldChange(field.key, e.target.checked)}
            className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
          />
        </div>
      );
    default:
      return null;
  }
}
