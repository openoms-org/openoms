"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import {
  RefreshCw,
  Plus,
  Trash2,
  Settings,
  Play,
  Pause,
  ArrowUpDown,
  AlertCircle,
  CheckCircle,
  Clock,
} from "lucide-react";
import { useTranslations } from "next-intl";
import {
  useListingSyncConfigs,
  useDeleteListingSyncConfig,
  useTriggerSync,
  useUpdateListingSyncConfig,
} from "@/hooks/use-listing-sync";
import { EmptyState } from "@/components/shared/empty-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { formatDate } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import type { ListingSyncConfig } from "@/types/api";

const STATUS_ICON: Record<string, React.ReactNode> = {
  active: <CheckCircle className="h-3.5 w-3.5" />,
  paused: <Pause className="h-3.5 w-3.5" />,
  error: <AlertCircle className="h-3.5 w-3.5" />,
};

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  active: "default",
  paused: "secondary",
  error: "destructive",
};

const STATUS_LABEL_KEYS: Record<string, string> = {
  active: "sync.statusActive",
  paused: "sync.statusPaused",
  error: "sync.statusError",
};

const DIRECTION_LABEL_KEYS: Record<string, string> = {
  push: "sync.directionPush",
  pull: "sync.directionPull",
  bidirectional: "sync.directionBidirectional",
};

const PRICE_RULE_LABEL_KEYS: Record<string, string> = {
  same: "sync.priceRuleSame",
  markup_pct: "sync.priceRuleMarkupPct",
  markup_fixed: "sync.priceRuleMarkupFixed",
  custom: "sync.priceRuleCustom",
};

function getProviderLabel(provider?: string, unknownLabel?: string): string {
  const labels: Record<string, string> = {
    allegro: "Allegro",
    amazon: "Amazon",
    ebay: "eBay",
    olx: "OLX",
    woocommerce: "WooCommerce",
    erli: "Erli",
  };
  return provider ? labels[provider] || provider : (unknownLabel ?? "Unknown");
}

export default function ListingSyncPage() {
  const t = useTranslations("listings");
  const router = useRouter();
  const [limit] = useState(20);
  const [offset] = useState(0);

  const { data, isLoading, isError, refetch } = useListingSyncConfigs({
    limit,
    offset,
  });
  const deleteConfig = useDeleteListingSyncConfig();

  const handleDelete = async (id: string) => {
    if (!confirm(t("sync.deleteConfirm"))) return;
    try {
      await deleteConfig.mutateAsync(id);
      toast.success(t("sync.deleteSuccess"));
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex flex-col items-center gap-4 py-12">
        <AlertCircle className="h-8 w-8 text-destructive" />
        <p className="text-muted-foreground">
          {t("sync.fetchError")}
        </p>
        <Button variant="outline" onClick={() => refetch()}>
          {t("sync.retry")}
        </Button>
      </div>
    );
  }

  const configs = data?.items || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("sync.title")}</h1>
          <p className="text-muted-foreground">
            {t("sync.subtitle")}
          </p>
        </div>
        <Link href="/listing-sync/new">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            {t("sync.newConfig")}
          </Button>
        </Link>
      </div>

      {configs.length === 0 ? (
        <EmptyState
          icon={ArrowUpDown}
          title={t("sync.emptyTitle")}
          description={t("sync.emptyDescription")}
          action={{
            label: t("sync.addConfig"),
            href: "/listing-sync/new",
          }}
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {configs.map((config) => (
            <ConfigCard
              key={config.id}
              config={config}
              onDelete={handleDelete}
              onNavigate={(id) => router.push(`/listing-sync/${id}`)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function ConfigCard({
  config,
  onDelete,
  onNavigate,
}: {
  config: ListingSyncConfig;
  onDelete: (id: string) => void;
  onNavigate: (id: string) => void;
}) {
  const t = useTranslations("listings");
  const triggerSync = useTriggerSync(config.id);
  const updateConfig = useUpdateListingSyncConfig(config.id);
  const [syncing, setSyncing] = useState(false);

  const statusIcon = STATUS_ICON[config.status] || STATUS_ICON.active;
  const statusVariant = STATUS_VARIANT[config.status] || STATUS_VARIANT.active;
  const statusLabelKey = STATUS_LABEL_KEYS[config.status] || STATUS_LABEL_KEYS.active;

  const handleSync = async () => {
    setSyncing(true);
    try {
      const result = await triggerSync.mutateAsync();
      toast.success(result.message);
    } catch (err) {
      toast.error(getErrorMessage(err));
    } finally {
      setSyncing(false);
    }
  };

  const handleToggleAutoSync = async (checked: boolean) => {
    try {
      await updateConfig.mutateAsync({ auto_sync: checked });
      toast.success(
        checked ? t("sync.autoSyncEnabled") : t("sync.autoSyncDisabled")
      );
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  return (
    <Card className="relative">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <CardTitle className="text-base">
              {getProviderLabel(config.integration_provider, t("sync.unknownProvider"))}
            </CardTitle>
            {config.integration_label && (
              <p className="text-sm text-muted-foreground">
                {config.integration_label}
              </p>
            )}
          </div>
          <Badge variant={statusVariant} className="gap-1">
            {statusIcon}
            {t(statusLabelKey)}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <span className="text-muted-foreground">{t("sync.direction")}</span>
            <p className="font-medium">
              {DIRECTION_LABEL_KEYS[config.sync_direction]
                ? t(DIRECTION_LABEL_KEYS[config.sync_direction])
                : config.sync_direction}
            </p>
          </div>
          <div>
            <span className="text-muted-foreground">{t("sync.priceRule")}</span>
            <p className="font-medium">
              {PRICE_RULE_LABEL_KEYS[config.price_rule]
                ? t(PRICE_RULE_LABEL_KEYS[config.price_rule])
                : config.price_rule}
              {config.price_modifier !== 0 && (
                <span className="ml-1 text-muted-foreground">
                  ({config.price_rule === "markup_pct" ? `${config.price_modifier}%` : `${config.price_modifier} PLN`})
                </span>
              )}
            </p>
          </div>
          {config.stock_buffer > 0 && (
            <div>
              <span className="text-muted-foreground">{t("sync.stockBuffer")}</span>
              <p className="font-medium">{config.stock_buffer} {t("sync.pieces")}</p>
            </div>
          )}
          <div>
            <span className="text-muted-foreground">{t("sync.interval")}</span>
            <p className="font-medium">{config.sync_interval_minutes} min</p>
          </div>
        </div>

        {config.last_sync_at && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            {t("sync.lastSync", { date: formatDate(config.last_sync_at) })}
          </div>
        )}

        {config.last_error && (
          <div className="flex items-start gap-1.5 rounded-md bg-destructive/10 p-2 text-xs text-destructive">
            <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            {config.last_error}
          </div>
        )}

        <div className="flex items-center justify-between border-t pt-3">
          <div className="flex items-center gap-2">
            <Switch
              checked={config.auto_sync}
              onCheckedChange={handleToggleAutoSync}
              disabled={updateConfig.isPending}
            />
            <span className="text-sm text-muted-foreground">Auto-sync</span>
          </div>
          <div className="flex items-center gap-1">
            <Button
              size="sm"
              variant="outline"
              onClick={handleSync}
              disabled={syncing}
            >
              {syncing ? (
                <RefreshCw className="mr-1 h-3.5 w-3.5 animate-spin" />
              ) : (
                <Play className="mr-1 h-3.5 w-3.5" />
              )}
              {t("sync.syncButton")}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => onNavigate(config.id)}
            >
              <Settings className="h-3.5 w-3.5" />
            </Button>
            <Button
              size="sm"
              variant="ghost"
              className="text-destructive hover:text-destructive"
              onClick={() => onDelete(config.id)}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
