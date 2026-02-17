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

const STATUS_CONFIG: Record<
  string,
  { label: string; icon: React.ReactNode; variant: "default" | "secondary" | "destructive" | "outline" }
> = {
  active: {
    label: "Aktywna",
    icon: <CheckCircle className="h-3.5 w-3.5" />,
    variant: "default",
  },
  paused: {
    label: "Wstrzymana",
    icon: <Pause className="h-3.5 w-3.5" />,
    variant: "secondary",
  },
  error: {
    label: "Blad",
    icon: <AlertCircle className="h-3.5 w-3.5" />,
    variant: "destructive",
  },
};

const DIRECTION_LABELS: Record<string, string> = {
  push: "Wyslij do marketplace",
  pull: "Pobierz z marketplace",
  bidirectional: "Dwukierunkowa",
};

const PRICE_RULE_LABELS: Record<string, string> = {
  same: "Bez zmian",
  markup_pct: "Narzut procentowy",
  markup_fixed: "Narzut stalowy",
  custom: "Niestandardowa",
};

function getProviderLabel(provider?: string): string {
  const labels: Record<string, string> = {
    allegro: "Allegro",
    amazon: "Amazon",
    ebay: "eBay",
    kaufland: "Kaufland",
    olx: "OLX",
    woocommerce: "WooCommerce",
    erli: "Erli",
    mirakl: "Mirakl/Empik",
  };
  return provider ? labels[provider] || provider : "Nieznany";
}

export default function ListingSyncPage() {
  const router = useRouter();
  const [limit] = useState(20);
  const [offset] = useState(0);

  const { data, isLoading, isError, refetch } = useListingSyncConfigs({
    limit,
    offset,
  });
  const deleteConfig = useDeleteListingSyncConfig();

  const handleDelete = async (id: string) => {
    if (!confirm("Czy na pewno chcesz usunac te konfiguracje synchronizacji?")) return;
    try {
      await deleteConfig.mutateAsync(id);
      toast.success("Konfiguracja usunieta");
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
          Nie udalo sie pobrac konfiguracji synchronizacji
        </p>
        <Button variant="outline" onClick={() => refetch()}>
          Ponow probe
        </Button>
      </div>
    );
  }

  const configs = data?.items || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Synchronizacja ofert</h1>
          <p className="text-muted-foreground">
            Zarzadzaj synchronizacja produktow i ofert z platformami marketplace
          </p>
        </div>
        <Link href="/listing-sync/new">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Nowa konfiguracja
          </Button>
        </Link>
      </div>

      {configs.length === 0 ? (
        <EmptyState
          icon={ArrowUpDown}
          title="Brak konfiguracji synchronizacji"
          description="Dodaj pierwsza konfiguracje, aby rozpoczac synchronizacje ofert z marketplace."
          action={{
            label: "Dodaj konfiguracje",
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
  const triggerSync = useTriggerSync(config.id);
  const updateConfig = useUpdateListingSyncConfig(config.id);
  const [syncing, setSyncing] = useState(false);

  const statusCfg = STATUS_CONFIG[config.status] || STATUS_CONFIG.active;

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
        checked ? "Auto-synchronizacja wlaczona" : "Auto-synchronizacja wylaczona"
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
              {getProviderLabel(config.integration_provider)}
            </CardTitle>
            {config.integration_label && (
              <p className="text-sm text-muted-foreground">
                {config.integration_label}
              </p>
            )}
          </div>
          <Badge variant={statusCfg.variant} className="gap-1">
            {statusCfg.icon}
            {statusCfg.label}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <span className="text-muted-foreground">Kierunek</span>
            <p className="font-medium">
              {DIRECTION_LABELS[config.sync_direction] || config.sync_direction}
            </p>
          </div>
          <div>
            <span className="text-muted-foreground">Regula ceny</span>
            <p className="font-medium">
              {PRICE_RULE_LABELS[config.price_rule] || config.price_rule}
              {config.price_modifier !== 0 && (
                <span className="ml-1 text-muted-foreground">
                  ({config.price_rule === "markup_pct" ? `${config.price_modifier}%` : `${config.price_modifier} PLN`})
                </span>
              )}
            </p>
          </div>
          {config.stock_buffer > 0 && (
            <div>
              <span className="text-muted-foreground">Bufor magazynowy</span>
              <p className="font-medium">{config.stock_buffer} szt.</p>
            </div>
          )}
          <div>
            <span className="text-muted-foreground">Interwat</span>
            <p className="font-medium">{config.sync_interval_minutes} min</p>
          </div>
        </div>

        {config.last_sync_at && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            Ostatnia synchronizacja: {formatDate(config.last_sync_at)}
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
              Synchronizuj
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
