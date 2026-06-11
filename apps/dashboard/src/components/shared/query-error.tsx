"use client";

import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";

interface QueryErrorProps {
  onRetry: () => void;
  messageOverride?: string;
}

export function QueryError({ onRetry, messageOverride }: QueryErrorProps) {
  const t = useTranslations("common");

  return (
    <div className="rounded-md border border-destructive bg-destructive/10 p-4">
      <p className="text-sm text-destructive">
        {messageOverride ?? t("loadError")}
      </p>
      <Button
        variant="outline"
        size="sm"
        className="mt-2"
        onClick={onRetry}
      >
        {t("retry")}
      </Button>
    </div>
  );
}
