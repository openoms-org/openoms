"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type {
  IntegrationHealthItem,
  OperationsHealth,
} from "@/lib/operations-dashboard";

interface IntegrationHealthPanelProps {
  items: IntegrationHealthItem[];
  isLoading: boolean;
}

const healthBadgeVariant: Record<OperationsHealth, "success" | "warning" | "destructive"> = {
  ok: "success",
  warning: "warning",
  problem: "destructive",
};

export function IntegrationHealthPanel({
  items,
  isLoading,
}: IntegrationHealthPanelProps) {
  const t = useTranslations("dashboard");

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("operations.integrationHealthTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-12 w-full" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            {t("operations.noIntegrationHealth")}
          </p>
        ) : (
          <div className="space-y-2">
            {items.map((item) => (
              <Link
                key={item.id}
                href={item.href}
                className="flex items-center justify-between gap-3 rounded-lg border bg-background p-3 transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <span className="min-w-0">
                  <span className="block truncate text-sm font-medium">{item.label}</span>
                  {item.errorMessage ? (
                    <span className="mt-1 block truncate text-xs text-muted-foreground">
                      {item.errorMessage}
                    </span>
                  ) : null}
                </span>
                <Badge variant={healthBadgeVariant[item.health]}>
                  {t(`operations.health.${item.health}`)}
                </Badge>
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
