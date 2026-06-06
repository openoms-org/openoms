"use client";

import { KeyRound, Eye } from "lucide-react";
import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import {
  buildTenantSetupPreview,
  type PreviewGroup,
} from "@/components/platform/providers/schema-builder/schema-helpers";
import type { ProviderFieldGroup } from "@/types/platform";

interface SetupPreviewProps {
  groups: ProviderFieldGroup[];
  environment?: "all" | "production" | "sandbox";
}

/**
 * Tenant setup-form preview generated from the schema. Shows only
 * customer-visible labels and help text, clearly distinguishes secret vs
 * non-secret, and never renders raw secret values (design spec §306-311).
 */
export function SetupPreview({ groups, environment = "all" }: SetupPreviewProps) {
  const t = useTranslations("platform");
  const preview: PreviewGroup[] = buildTenantSetupPreview(groups, environment);

  if (preview.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        {t("schemaBuilder.preview.empty")}
      </p>
    );
  }

  return (
    <div className="space-y-5" data-testid="setup-preview">
      {preview.map((group) => (
        <div key={group.key} className="space-y-3">
          <h4 className="text-sm font-semibold">{group.label}</h4>
          <div className="space-y-3">
            {group.fields.map((field) => (
              <div key={field.key} className="grid gap-1.5">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">{field.label}</span>
                  {field.required && (
                    <span className="text-destructive" aria-hidden>
                      *
                    </span>
                  )}
                  {field.secret ? (
                    <Badge variant="warning" className="gap-1">
                      <KeyRound className="h-3 w-3" />
                      {t("schemaBuilder.preview.secret")}
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="gap-1">
                      <Eye className="h-3 w-3" />
                      {t("schemaBuilder.preview.visible")}
                    </Badge>
                  )}
                  {field.environmentScope && field.environmentScope !== "all" && (
                    <Badge variant="secondary">
                      {t(`schemaBuilder.envScope.${field.environmentScope}`)}
                    </Badge>
                  )}
                </div>
                {field.type === "enum" && field.enumValues.length > 0 ? (
                  <div className="flex flex-wrap gap-1.5">
                    {field.enumValues.map((value) => (
                      <Badge key={value} variant="outline">
                        {value}
                      </Badge>
                    ))}
                  </div>
                ) : (
                  <div className="h-9 rounded-md border border-dashed bg-muted/30 px-3 text-sm leading-9 text-muted-foreground">
                    {field.secret
                      ? "••••••••"
                      : t(`schemaBuilder.fieldType.${field.type}`)}
                  </div>
                )}
                {field.helpText && (
                  <p className="text-xs text-muted-foreground">{field.helpText}</p>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
