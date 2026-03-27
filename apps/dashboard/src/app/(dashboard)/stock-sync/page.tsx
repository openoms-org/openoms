"use client";

import { useState } from "react";
import {
  RefreshCw,
  Plus,
  Trash2,
  Activity,
  Package,
  AlertCircle,
  CheckCircle2,
  Clock,
  XCircle,
} from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { EmptyState } from "@/components/shared/empty-state";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
} from "@/components/ui/dialog";
import { Switch } from "@/components/ui/switch";
import {
  useStockSyncChannels,
  useStockSyncDashboard,
  useCreateStockSyncChannel,
  useUpdateStockSyncChannel,
  useDeleteStockSyncChannel,
  usePushAllStock,
  usePushChannelStock,
} from "@/hooks/use-stock-sync";
import type {
  StockSyncChannel,
  ChannelSummary,
  CreateStockSyncChannelRequest,
} from "@/types/api";
import Link from "next/link";
import { useTranslations } from "next-intl";

function useChannelTypeLabels() {
  const t = useTranslations("stockSync");
  return {
    allegro: "Allegro",
    amazon: "Amazon",
    woocommerce: "WooCommerce",
    ebay: "eBay",
    shopify: "Shopify",
    manual: t("channelManual"),
  } as Record<string, string>;
}

function useSyncModeLabels() {
  const t = useTranslations("stockSync");
  return {
    realtime: t("syncModeRealtime"),
    scheduled: t("syncModeScheduled"),
    manual: t("syncModeManual"),
  } as Record<string, string>;
}

function ChannelStatusIndicator({ status }: { status: string }) {
  switch (status) {
    case "ok":
      return <CheckCircle2 className="h-5 w-5 text-green-500" />;
    case "warning":
      return <Clock className="h-5 w-5 text-yellow-500" />;
    case "error":
      return <XCircle className="h-5 w-5 text-red-500" />;
    case "disabled":
      return <XCircle className="h-5 w-5 text-gray-400" />;
    default:
      return <Clock className="h-5 w-5 text-gray-400" />;
  }
}

function StatusBadge({ status }: { status: string }) {
  const t = useTranslations("stockSync");
  const variants: Record<string, string> = {
    ok: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
    warning:
      "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
    error: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
    disabled: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
  };
  const labels: Record<string, string> = {
    ok: "OK",
    warning: t("noSync"),
    error: t("invoice.error"),
    disabled: t("disabled"),
  };

  return (
    <Badge variant="outline" className={variants[status] || ""}>
      {labels[status] || status}
    </Badge>
  );
}

function AddChannelDialog() {
  const t = useTranslations("stockSync");
  const channelTypeLabels = useChannelTypeLabels();
  const syncModeLabels = useSyncModeLabels();
  const [open, setOpen] = useState(false);
  const [channelType, setChannelType] = useState("allegro");
  const [syncMode, setSyncMode] = useState("realtime");
  const [stockBuffer, setStockBuffer] = useState(0);
  const [priority, setPriority] = useState(0);

  const createChannel = useCreateStockSyncChannel();

  const handleSubmit = async () => {
    const req: CreateStockSyncChannelRequest = {
      channel_type: channelType,
      sync_mode: syncMode,
      stock_buffer: stockBuffer,
      priority: priority,
      enabled: true,
    };
    await createChannel.mutateAsync(req);
    setOpen(false);
    setChannelType("allegro");
    setSyncMode("realtime");
    setStockBuffer(0);
    setPriority(0);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          {t("addChannel")}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("newSyncChannel")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label>{t("channelType")}</Label>
            <Select value={channelType} onValueChange={setChannelType}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(channelTypeLabels).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>{t("syncMode")}</Label>
            <Select value={syncMode} onValueChange={setSyncMode}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(syncModeLabels).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>{t("buforBezpieczenstwaSzt")}</Label>
            <Input
              type="number"
              min={0}
              value={stockBuffer}
              onChange={(e) => setStockBuffer(Number(e.target.value))}
            />
            <p className="text-xs text-muted-foreground mt-1">
              {t("iloscSztukRezerwowanaJakoZapasBezpieczenstwaNa")}
            </p>
          </div>
          <div>
            <Label>{t("priority")}</Label>
            <Input
              type="number"
              value={priority}
              onChange={(e) => setPriority(Number(e.target.value))}
            />
            <p className="text-xs text-muted-foreground mt-1">
              {t("priorityHint")}
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            {t("cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={createChannel.isPending}>
            {createChannel.isPending ? t("creating") : t("create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ChannelCard({ channel }: { channel: ChannelSummary }) {
  const t = useTranslations("stockSync");
  const channelTypeLabels = useChannelTypeLabels();
  const syncModeLabels = useSyncModeLabels();
  const deleteChannel = useDeleteStockSyncChannel();
  const updateChannel = useUpdateStockSyncChannel(channel.id);
  const pushChannel = usePushChannelStock();

  const handleToggle = async (enabled: boolean) => {
    await updateChannel.mutateAsync({ enabled });
  };

  const handleDelete = async () => {
    if (confirm(t("confirmDeleteChannel"))) {
      await deleteChannel.mutateAsync(channel.id);
    }
  };

  const handlePush = async () => {
    await pushChannel.mutateAsync(channel.id);
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <div className="flex items-center gap-2">
          <ChannelStatusIndicator status={channel.status} />
          <CardTitle className="text-base">
            {channelTypeLabels[channel.channel_type] || channel.channel_type}
          </CardTitle>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={handlePush}
            disabled={pushChannel.isPending || !channel.enabled}
            title={t("syncChannel")}
          >
            <RefreshCw
              className={`h-4 w-4 text-muted-foreground ${pushChannel.isPending ? "animate-spin" : ""}`}
            />
          </Button>
          <Switch
            checked={channel.enabled}
            onCheckedChange={handleToggle}
            aria-label={t("enableDisableChannel")}
          />
          <Button
            variant="ghost"
            size="icon"
            onClick={handleDelete}
            disabled={deleteChannel.isPending}
          >
            <Trash2 className="h-4 w-4 text-muted-foreground" />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("status")}</span>
            <StatusBadge status={channel.status} />
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("mode")}</span>
            <span>{syncModeLabels[channel.sync_mode] || channel.sync_mode}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("buffer")}</span>
            <span>{channel.stock_buffer} {t("pcs")}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("errors24h")}</span>
            <span className={channel.error_count > 0 ? "text-red-600 font-medium" : ""}>
              {channel.error_count}
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("zsyncProduktow")}</span>
            <span>{channel.items_synced}</span>
          </div>
          {channel.last_sync_at && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("lastSync")}</span>
              <span>{formatDate(channel.last_sync_at)}</span>
            </div>
          )}
          {channel.last_error && (
            <div className="mt-2 rounded-md bg-red-50 dark:bg-red-950 p-2 text-xs text-red-700 dark:text-red-300">
              {channel.last_error}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

export default function StockSyncPage() {
  const t = useTranslations("stockSync");
  const { data: dashboard, isLoading: dashLoading, dataUpdatedAt } = useStockSyncDashboard();
  const { isLoading: channelsLoading } =
    useStockSyncChannels({ limit: 100 });
  const pushAll = usePushAllStock();

  const isLoading = dashLoading || channelsLoading;

  return (
    <AdminGuard>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {t("stockSync")}
            </h1>
            <p className="text-muted-foreground">
              {t("stockSyncWithSalesChannels")}
              rzeczywistym
            </p>
            {dataUpdatedAt > 0 && (
              <p className="text-xs text-muted-foreground mt-1">
                {t("refreshedAt")}: {new Date(dataUpdatedAt).toLocaleTimeString()}
              </p>
            )}
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => pushAll.mutate()}
              disabled={pushAll.isPending}
            >
              <RefreshCw
                className={`mr-2 h-4 w-4 ${pushAll.isPending ? "animate-spin" : ""}`}
              />
              {pushAll.isPending ? t("synchronizuje") : t("syncAll")}
            </Button>
            <AddChannelDialog />
          </div>
        </div>

        {isLoading ? (
          <LoadingSkeleton />
        ) : (
          <>
            {/* Stats cards */}
            <div className="grid gap-4 md:grid-cols-4">
              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    {t("productsWithStock")}
                  </CardTitle>
                  <Package className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">
                    {dashboard?.total_products ?? 0}
                  </div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    {t("activeChannels")}
                  </CardTitle>
                  <Activity className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">
                    {dashboard?.active_channels ?? 0}
                  </div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    {t("errors24h")}
                  </CardTitle>
                  <AlertCircle className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div
                    className={`text-2xl font-bold ${
                      (dashboard?.recent_errors ?? 0) > 0
                        ? "text-red-600"
                        : ""
                    }`}
                  >
                    {dashboard?.recent_errors ?? 0}
                  </div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    {t("lastSync")}
                  </CardTitle>
                  <Clock className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-sm font-medium">
                    {dashboard?.last_sync_at
                      ? formatDate(dashboard.last_sync_at)
                      : t("never")}
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Channels grid */}
            {dashboard?.channel_summaries &&
            dashboard.channel_summaries.length > 0 ? (
              <div>
                <h2 className="text-lg font-semibold mb-4">{t("syncChannels")}</h2>
                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                  {dashboard.channel_summaries.map((ch) => (
                    <ChannelCard key={ch.id} channel={ch} />
                  ))}
                </div>
              </div>
            ) : (
              <EmptyState
                icon={RefreshCw}
                title={t("noSyncChannels")}
                description={t("addSalesChannelToStartSync")}
              />
            )}

            {/* Link to events log */}
            <div className="flex justify-end">
              <Link href="/stock-sync/events">
                <Button variant="outline">
                  {t("historiaZdarzen")}
                </Button>
              </Link>
            </div>
          </>
        )}
      </div>
    </AdminGuard>
  );
}
