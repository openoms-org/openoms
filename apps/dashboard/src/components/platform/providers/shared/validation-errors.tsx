"use client";

import { AlertTriangle, Info } from "lucide-react";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import type { FieldValidationIssue } from "@/components/platform/providers/schema-builder/schema-helpers";

interface ValidationErrorsProps {
  issues: FieldValidationIssue[];
  className?: string;
}

/**
 * Renders field validation issues. Blocking issues use a destructive tone;
 * advisory warnings (e.g. "should be secret") use a muted warning tone. Each
 * code maps to a `schemaBuilder.validation.<code>` message.
 */
export function ValidationErrors({ issues, className }: ValidationErrorsProps) {
  const t = useTranslations("platform");

  if (issues.length === 0) {
    return null;
  }

  return (
    <ul className={cn("space-y-1", className)} role="list">
      {issues.map((issue) => {
        const Icon = issue.warning ? Info : AlertTriangle;
        return (
          <li
            key={issue.code}
            className={cn(
              "flex items-start gap-1.5 text-xs",
              issue.warning ? "text-warning" : "text-destructive",
            )}
          >
            <Icon className="mt-0.5 h-3 w-3 shrink-0" />
            <span>{t(`schemaBuilder.validation.${issue.code}`)}</span>
          </li>
        );
      })}
    </ul>
  );
}
