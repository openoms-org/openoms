"use client";

import type { ReactNode } from "react";
import { useTranslations } from "next-intl";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  useForm,
  type DefaultValues,
  type FieldError,
  type FieldErrors,
  type FieldValues,
  type Path,
  type Resolver,
  type SubmitHandler,
  type UseFormReturn,
} from "react-hook-form";
import type { z } from "zod";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

interface FormWrapperProps<TValues extends FieldValues> {
  schema: z.ZodType<TValues, FieldValues>;
  defaultValues: DefaultValues<TValues>;
  onSubmit: SubmitHandler<TValues>;
  children: (form: UseFormReturn<TValues>) => ReactNode;
  className?: string;
  actionsClassName?: string;
  submitLabel: ReactNode;
  submittingLabel?: ReactNode;
  cancelLabel?: ReactNode;
  isSubmitting?: boolean;
  onCancel?: () => void;
  hideActions?: boolean;
  showErrorSummary?: boolean;
  errorSummaryTitle?: ReactNode;
}

export function FormWrapper<TValues extends FieldValues>({
  schema,
  defaultValues,
  onSubmit,
  children,
  className,
  actionsClassName,
  submitLabel,
  submittingLabel,
  cancelLabel = "Cancel",
  isSubmitting = false,
  onCancel,
  hideActions = false,
  showErrorSummary = true,
  errorSummaryTitle = "Please fix the highlighted fields",
}: FormWrapperProps<TValues>) {
  const t = useTranslations("common");
  const form = useForm<TValues>({
    resolver: zodResolver(schema) as Resolver<TValues>,
    defaultValues,
  });
  const pending = isSubmitting || form.formState.isSubmitting;
  const errorMessages = collectErrorMessages(form.formState.errors);

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className={className} noValidate>
      {showErrorSummary && errorMessages.length > 0 && (
        <div
          role="alert"
          className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive"
        >
          <p className="font-medium">{errorSummaryTitle}</p>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            {errorMessages.map((message, index) => (
              <li key={`${message}-${index}`}>{message}</li>
            ))}
          </ul>
        </div>
      )}

      {children(form)}

      {!hideActions && (
        <div className={cn("flex items-center gap-3", actionsClassName)}>
          <Button type="submit" disabled={pending}>
            {pending ? (submittingLabel ?? t("processing")) : submitLabel}
          </Button>
          {onCancel && (
            <Button type="button" variant="outline" onClick={onCancel}>
              {cancelLabel}
            </Button>
          )}
        </div>
      )}
    </form>
  );
}

interface FormFieldProps<TValues extends FieldValues> {
  name?: Path<TValues>;
  label: ReactNode;
  children: ReactNode;
  error?: FieldError;
  description?: ReactNode;
  required?: boolean;
  htmlFor?: string;
  className?: string;
  labelClassName?: string;
  errorClassName?: string;
}

export function FormField<TValues extends FieldValues>({
  name,
  label,
  children,
  error,
  description,
  required = false,
  htmlFor,
  className,
  labelClassName,
  errorClassName,
}: FormFieldProps<TValues>) {
  const fieldId = htmlFor ?? name;

  return (
    <div className={cn("space-y-2", className)}>
      <Label htmlFor={fieldId} className={labelClassName}>
        {label}
        {required && (
          <span aria-hidden="true" className="text-destructive ml-1">
            *
          </span>
        )}
      </Label>
      {children}
      {description && <p className="text-sm text-muted-foreground">{description}</p>}
      {error?.message && (
        <p className={cn("text-destructive text-xs mt-1", errorClassName)}>
          {error.message}
        </p>
      )}
    </div>
  );
}

function collectErrorMessages(errors: FieldErrors<FieldValues>): string[] {
  const messages: string[] = [];

  for (const value of Object.values(errors)) {
    if (!value) continue;

    if (Array.isArray(value)) {
      for (const item of value) {
        if (item && typeof item === "object") {
          messages.push(...collectErrorMessages(item as FieldErrors<FieldValues>));
        }
      }
      continue;
    }

    if (typeof value === "object") {
      if ("message" in value && typeof value.message === "string") {
        messages.push(value.message);
        continue;
      }
      messages.push(...collectErrorMessages(value as FieldErrors<FieldValues>));
    }
  }

  return Array.from(new Set(messages));
}
