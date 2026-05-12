"use client";

import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { enUS, pl } from "date-fns/locale";
import { useLocale, useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { ProviderLogo } from "@/components/shared/provider-logo";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
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
  const locale = useLocale();
  const dateLocale = locale === "pl" ? pl : enUS;

  return (
    <Card className="overflow-hidden rounded-lg border-border/80 bg-card py-0 shadow-sm">
      <CardHeader className="border-b bg-background/70 px-5 py-4">
        <CardTitle className="text-base">{t("operations.integrationHealthTitle")}</CardTitle>
      </CardHeader>
      <CardContent className="p-5">
        {isLoading ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-32 w-full rounded-lg" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <div className="flex min-h-32 items-center justify-center rounded-lg border border-dashed bg-background/70 px-6 text-center">
            <p className="text-sm text-muted-foreground">
              {t("operations.noIntegrationHealth")}
            </p>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            {items.map((item) => (
              <Link
                key={item.id}
                href={item.href}
                className={cn(
                  "flex min-h-32 flex-col justify-between rounded-lg border bg-background p-4 transition-colors hover:border-primary/40 hover:bg-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                  item.health === "problem" && "border-destructive/40",
                  item.health === "warning" && "border-warning/50",
                )}
              >
                <span className="flex items-start justify-between gap-3">
                  <span className="flex min-w-0 items-center gap-3">
                    <ProviderLogo providerKey={item.provider} size="md" />
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-semibold">{item.label}</span>
                      <span className="mt-1 block text-xs text-muted-foreground">
                        {item.lastSyncAt
                          ? t("operations.lastSync", {
                              time: formatDistanceToNow(new Date(item.lastSyncAt), {
                                addSuffix: true,
                                locale: dateLocale,
                              }),
                            })
                          : t("operations.noLastSync")}
                      </span>
                    </span>
                  </span>
                  <Badge variant={healthBadgeVariant[item.health]}>
                    {t(`operations.health.${item.health}`)}
                  </Badge>
                </span>
                {item.errorMessage ? (
                  <span className="mt-3 line-clamp-2 text-xs text-muted-foreground">
                    {item.errorMessage}
                  </span>
                ) : (
                  <span className="mt-3 text-xs text-muted-foreground">
                    {t("operations.connectionReady")}
                  </span>
                )}
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
