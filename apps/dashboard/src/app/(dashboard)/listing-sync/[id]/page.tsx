"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import {
  ArrowLeft,
  Loader2,
  RefreshCw,
  Play,
  DollarSign,
  Package,
  AlertCircle,
  CheckCircle,
  XCircle,
  SkipForward,
  Clock,
  Save,
} from "lucide-react";
import {
  useListingSyncConfig,
  useUpdateListingSyncConfig,
  useTriggerSync,
  useTriggerSyncPrices,
  useTriggerSyncStock,
  useListingSyncLogs,
} from "@/hooks/use-listing-sync";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { formatDate } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import type { ListingSyncLog } from "@/types/api";

const LOG_STATUS_CONFIG: Record<
  string,
  { label: string; icon: React.ReactNode; className: string }
> = {
  success: {
    label: "Sukces",
    icon: <CheckCircle className="h-3.5 w-3.5" />,
    className: "text-green-700 dark:text-green-400",
  },
  failed: {
    label: "Blad",
    icon: <XCircle className="h-3.5 w-3.5" />,
    className: "text-red-700 dark:text-red-400",
  },
  skipped: {
    label: "Pominieto",
    icon: <SkipForward className="h-3.5 w-3.5" />,
    className: "text-yellow-700 dark:text-yellow-400",
  },
};

const ENTITY_TYPE_LABELS: Record<string, string> = {
  product: "Produkt",
  price: "Cena",
  stock: "Stan magazynowy",
  offer: "Oferta",
};

const DIRECTION_LABELS: Record<string, string> = {
  push: "Wyslij",
  pull: "Pobierz",
};

export default function ListingSyncDetailPage() {
  const params = useParams();
  const configId = params.id as string;

  const { data: config, isLoading, isError } = useListingSyncConfig(configId);
  const updateConfig = useUpdateListingSyncConfig(configId);
  const triggerSync = useTriggerSync(configId);
  const triggerSyncPrices = useTriggerSyncPrices(configId);
  const triggerSyncStock = useTriggerSyncStock(configId);

  // Edit form state
  const [syncDirection, setSyncDirection] = useState<string>("");
  const [priceRule, setPriceRule] = useState<string>("");
  const [priceModifier, setPriceModifier] = useState<number>(0);
  const [stockBuffer, setStockBuffer] = useState<number>(0);
  const [syncInterval, setSyncInterval] = useState<number>(30);
  const [formDirty, setFormDirty] = useState(false);

  // Log pagination
  const [logLimit, setLogLimit] = useState(10);
  const [logOffset, setLogOffset] = useState(0);
  const { data: logsData, isLoading: loadingLogs } = useListingSyncLogs(
    configId,
    { limit: logLimit, offset: logOffset }
  );

  // Initialize form when config loads
  const initializeForm = () => {
    if (config && !formDirty) {
      setSyncDirection(config.sync_direction);
      setPriceRule(config.price_rule);
      setPriceModifier(config.price_modifier);
      setStockBuffer(config.stock_buffer);
      setSyncInterval(config.sync_interval_minutes);
    }
  };

  // Run init on each render (when config is loaded)
  if (config && syncDirection === "") {
    initializeForm();
  }

  const handleSave = async () => {
    try {
      await updateConfig.mutateAsync({
        sync_direction: syncDirection as "push" | "pull" | "bidirectional",
        price_rule: priceRule as "same" | "markup_pct" | "markup_fixed" | "custom",
        price_modifier: priceModifier,
        stock_buffer: stockBuffer,
        sync_interval_minutes: syncInterval,
      });
      setFormDirty(false);
      toast.success("Konfiguracja zapisana");
    } catch (err) {
      toast.error(getErrorMessage(err));
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

  const handleToggleStatus = async () => {
    if (!config) return;
    const newStatus = config.status === "active" ? "paused" : "active";
    try {
      await updateConfig.mutateAsync({ status: newStatus });
      toast.success(
        newStatus === "active"
          ? "Synchronizacja aktywowana"
          : "Synchronizacja wstrzymana"
      );
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  const handleFullSync = async () => {
    try {
      const result = await triggerSync.mutateAsync();
      toast.success(result.message);
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  const handleSyncPrices = async () => {
    try {
      const result = await triggerSyncPrices.mutateAsync();
      toast.success(result.message);
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  const handleSyncStock = async () => {
    try {
      const result = await triggerSyncStock.mutateAsync();
      toast.success(result.message);
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

  if (isError || !config) {
    return (
      <div className="flex flex-col items-center gap-4 py-12">
        <AlertCircle className="h-8 w-8 text-destructive" />
        <p className="text-muted-foreground">
          Konfiguracja synchronizacji nie zostala znaleziona
        </p>
        <Link href="/listing-sync">
          <Button variant="outline">Powrot do listy</Button>
        </Link>
      </div>
    );
  }

  const logs = logsData?.items || [];
  const totalLogs = logsData?.total || 0;
  const syncing =
    triggerSync.isPending ||
    triggerSyncPrices.isPending ||
    triggerSyncStock.isPending;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/listing-sync">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-bold">
            Synchronizacja:{" "}
            {config.integration_provider
              ? config.integration_provider.charAt(0).toUpperCase() +
                config.integration_provider.slice(1)
              : "Nieznana"}
          </h1>
          {config.integration_label && (
            <p className="text-muted-foreground">{config.integration_label}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Badge
            variant={config.status === "active" ? "default" : config.status === "error" ? "destructive" : "secondary"}
          >
            {config.status === "active" ? "Aktywna" : config.status === "error" ? "Blad" : "Wstrzymana"}
          </Badge>
          <Button variant="outline" size="sm" onClick={handleToggleStatus}>
            {config.status === "active" ? "Wstrzymaj" : "Aktywuj"}
          </Button>
        </div>
      </div>

      {/* Sync Actions */}
      <Card>
        <CardHeader>
          <CardTitle>Akcje synchronizacji</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-3">
            <Button onClick={handleFullSync} disabled={syncing}>
              {triggerSync.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Play className="mr-2 h-4 w-4" />
              )}
              Pelna synchronizacja
            </Button>
            <Button variant="outline" onClick={handleSyncPrices} disabled={syncing}>
              {triggerSyncPrices.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <DollarSign className="mr-2 h-4 w-4" />
              )}
              Synchronizuj ceny
            </Button>
            <Button variant="outline" onClick={handleSyncStock} disabled={syncing}>
              {triggerSyncStock.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Package className="mr-2 h-4 w-4" />
              )}
              Synchronizuj stany
            </Button>
          </div>

          {config.last_sync_at && (
            <div className="mt-3 flex items-center gap-1.5 text-sm text-muted-foreground">
              <Clock className="h-4 w-4" />
              Ostatnia synchronizacja: {formatDate(config.last_sync_at)}
            </div>
          )}
          {config.last_error && (
            <div className="mt-2 flex items-start gap-1.5 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
              {config.last_error}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Configuration */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Ustawienia synchronizacji</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Kierunek synchronizacji</Label>
              <Select
                value={syncDirection}
                onValueChange={(v) => {
                  setSyncDirection(v);
                  setFormDirty(true);
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="bidirectional">Dwukierunkowa</SelectItem>
                  <SelectItem value="push">Wyslij do marketplace</SelectItem>
                  <SelectItem value="pull">Pobierz z marketplace</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Regula ceny</Label>
              <Select
                value={priceRule}
                onValueChange={(v) => {
                  setPriceRule(v);
                  setFormDirty(true);
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="same">Bez zmian</SelectItem>
                  <SelectItem value="markup_pct">Narzut procentowy</SelectItem>
                  <SelectItem value="markup_fixed">Narzut stalowy (PLN)</SelectItem>
                  <SelectItem value="custom">Niestandardowa</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {(priceRule === "markup_pct" || priceRule === "markup_fixed") && (
              <div className="space-y-2">
                <Label>
                  {priceRule === "markup_pct"
                    ? "Narzut (%)"
                    : "Narzut (PLN)"}
                </Label>
                <Input
                  type="number"
                  step={priceRule === "markup_pct" ? "0.1" : "0.01"}
                  value={priceModifier}
                  onChange={(e) => {
                    setPriceModifier(parseFloat(e.target.value) || 0);
                    setFormDirty(true);
                  }}
                />
              </div>
            )}

            <div className="space-y-2">
              <Label>Bufor magazynowy (szt.)</Label>
              <Input
                type="number"
                min={0}
                value={stockBuffer}
                onChange={(e) => {
                  setStockBuffer(parseInt(e.target.value) || 0);
                  setFormDirty(true);
                }}
              />
              <p className="text-xs text-muted-foreground">
                Zapas bezpieczenstwa odejmowany od dostepnego stanu
              </p>
            </div>

            {formDirty && (
              <Button onClick={handleSave} disabled={updateConfig.isPending}>
                {updateConfig.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Save className="mr-2 h-4 w-4" />
                )}
                Zapisz zmiany
              </Button>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Automatyzacja</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Automatyczna synchronizacja</Label>
                <p className="text-sm text-muted-foreground">
                  Synchronizuj automatycznie w ustalonym interwale
                </p>
              </div>
              <Switch
                checked={config.auto_sync}
                onCheckedChange={handleToggleAutoSync}
                disabled={updateConfig.isPending}
              />
            </div>

            <div className="space-y-2">
              <Label>Interwal (minuty)</Label>
              <Input
                type="number"
                min={5}
                max={1440}
                value={syncInterval}
                onChange={(e) => {
                  setSyncInterval(parseInt(e.target.value) || 30);
                  setFormDirty(true);
                }}
              />
            </div>

            <div className="rounded-md border p-3 text-sm">
              <h4 className="font-medium">Mapowanie pol</h4>
              <p className="mt-1 text-muted-foreground">
                Domyslnie synchronizowane sa: nazwa, opis, cena, stan
                magazynowy, SKU, EAN i zdjecia. Niestandardowe mapowanie pol
                zostanie dodane w przyszlych wersjach.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Sync Log History */}
      <Card>
        <CardHeader>
          <CardTitle>Dziennik synchronizacji</CardTitle>
        </CardHeader>
        <CardContent>
          {loadingLogs ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : logs.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">
              Brak wpisow w dzienniku synchronizacji. Uruchom pierwsza
              synchronizacje.
            </p>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Data</TableHead>
                    <TableHead>Kierunek</TableHead>
                    <TableHead>Typ</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>ID zewnetrzne</TableHead>
                    <TableHead>Szczegoly</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map((log) => (
                    <LogRow key={log.id} log={log} />
                  ))}
                </TableBody>
              </Table>

              {totalLogs > logLimit && (
                <div className="mt-4">
                  <DataTablePagination
                    total={totalLogs}
                    limit={logLimit}
                    offset={logOffset}
                    onPageChange={setLogOffset}
                    onPageSizeChange={(newLimit) => {
                      setLogLimit(newLimit);
                      setLogOffset(0);
                    }}
                  />
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function LogRow({ log }: { log: ListingSyncLog }) {
  const statusCfg = LOG_STATUS_CONFIG[log.status] || LOG_STATUS_CONFIG.success;

  return (
    <TableRow>
      <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
        {formatDate(log.created_at)}
      </TableCell>
      <TableCell>
        <Badge variant="outline" className="text-xs">
          {DIRECTION_LABELS[log.direction] || log.direction}
        </Badge>
      </TableCell>
      <TableCell className="text-sm">
        {ENTITY_TYPE_LABELS[log.entity_type] || log.entity_type}
      </TableCell>
      <TableCell>
        <span className={`flex items-center gap-1 text-xs ${statusCfg.className}`}>
          {statusCfg.icon}
          {statusCfg.label}
        </span>
      </TableCell>
      <TableCell className="text-xs text-muted-foreground font-mono">
        {log.external_id || "-"}
      </TableCell>
      <TableCell className="max-w-xs truncate text-xs text-muted-foreground">
        {log.error_message
          ? log.error_message
          : log.changes
          ? JSON.stringify(log.changes).slice(0, 80)
          : "-"}
      </TableCell>
    </TableRow>
  );
}
