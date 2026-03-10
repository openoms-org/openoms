"use client";

import { useTranslations } from "next-intl";
import { AlertTriangle } from "lucide-react";
import { Badge } from "@/components/ui/badge";

/**
 * Full-width amber banner displayed at the top of pages for integrations
 * that have not yet been verified in a production environment.
 */
export function DevelopmentBanner() {
  const t = useTranslations("shared.developmentBanner");

  return (
    <div className="flex items-start gap-3 rounded-md border border-warning/30 bg-warning/10 p-4">
      <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-warning" />
      <div className="text-sm text-warning">
        <strong>{t("title")}</strong> &mdash; {t("message")}
      </div>
    </div>
  );
}

/**
 * Small inline badge reading "Under construction", intended for use in tables and lists
 * next to integration provider names that are not yet production-verified.
 */
export function DevelopmentBadge() {
  const t = useTranslations("shared.developmentBanner");

  return (
    <Badge variant="warning" className="ml-2 text-[10px] leading-tight">
      {t("badge")}
    </Badge>
  );
}
