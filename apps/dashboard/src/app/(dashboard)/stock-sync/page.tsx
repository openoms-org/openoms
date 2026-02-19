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
} from "@/hooks/use-stock-sync";
import type {
  StockSyncChannel,
  ChannelSummary,
  CreateStockSyncChannelRequest,
} from "@/types/api";
import Link from "next/link";

const channelTypeLabels: Record<string, string> = {
  allegro: "Allegro",
  amazon: "Amazon",
  woocommerce: "WooCommerce",
  ebay: "eBay",
  shopify: "Shopify",
  manual: "Ręczny",
};

const syncModeLabels: Record<string, string> = {
  realtime: "Czas rzeczywisty",
  scheduled: "Zaplanowany",
  manual: "Ręczny",
};

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
  const variants: Record<string, string> = {
    ok: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
    warning:
      "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
    error: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
    disabled: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
  };
  const labels: Record<string, string> = {
    ok: "OK",
    warning: "Brak synchronizacji",
    error: "Błąd",
    disabled: "Wyłączony",
  };

  return (
    <Badge variant="outline" className={variants[status] || ""}>
      {labels[status] || status}
    </Badge>
  );
}

function AddChannelDialog() {
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
          Dodaj kanał
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Nowy kanał synchronizacji</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label>Typ kanału</Label>
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
            <Label>Tryb synchronizacji</Label>
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
            <Label>Bufor bezpieczeństwa (szt.)</Label>
            <Input
              type="number"
              min={0}
              value={stockBuffer}
              onChange={(e) => setStockBuffer(Number(e.target.value))}
            />
            <p className="text-xs text-muted-foreground mt-1">
              Ilość sztuk rezerwowana jako zapas bezpieczeństwa na tym kanale
            </p>
          </div>
          <div>
            <Label>Priorytet</Label>
            <Input
              type="number"
              value={priority}
              onChange={(e) => setPriority(Number(e.target.value))}
            />
            <p className="text-xs text-muted-foreground mt-1">
              Wyższy priorytet = pierwszeństwo przy ograniczonym stanie
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Anuluj
          </Button>
          <Button onClick={handleSubmit} disabled={createChannel.isPending}>
            {createChannel.isPending ? "Tworzenie..." : "Utwórz"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ChannelCard({ channel }: { channel: ChannelSummary }) {
  const deleteChannel = useDeleteStockSyncChannel();
  const updateChannel = useUpdateStockSyncChannel(channel.id);

  const handleToggle = async (enabled: boolean) => {
    await updateChannel.mutateAsync({ enabled });
  };

  const handleDelete = async () => {
    if (confirm("Czy na pewno chcesz usunąć ten kanał?")) {
      await deleteChannel.mutateAsync(channel.id);
    }
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
          <Switch
            checked={channel.enabled}
            onCheckedChange={handleToggle}
            aria-label="Włącz/wyłącz kanał"
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
            <span className="text-muted-foreground">Status</span>
            <StatusBadge status={channel.status} />
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Tryb</span>
            <span>{syncModeLabels[channel.sync_mode] || channel.sync_mode}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Bufor</span>
            <span>{channel.stock_buffer} szt.</span>
          </div>
          {channel.last_sync_at && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">Ostatnia synchronizacja</span>
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
  const { data: dashboard, isLoading: dashLoading } = useStockSyncDashboard();
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
              Sync magazynu
            </h1>
            <p className="text-muted-foreground">
              Synchronizacja stanów magazynowych z kanałami sprzedaży w czasie
              rzeczywistym
            </p>
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
              {pushAll.isPending ? "Synchronizuję..." : "Synchronizuj wszystko"}
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
                    Produkty z magazynem
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
                    Aktywne kanały
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
                    Błędy (24h)
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
                    Ostatnia synchronizacja
                  </CardTitle>
                  <Clock className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-sm font-medium">
                    {dashboard?.last_sync_at
                      ? formatDate(dashboard.last_sync_at)
                      : "Nigdy"}
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Channels grid */}
            {dashboard?.channel_summaries &&
            dashboard.channel_summaries.length > 0 ? (
              <div>
                <h2 className="text-lg font-semibold mb-4">Kanały synchronizacji</h2>
                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                  {dashboard.channel_summaries.map((ch) => (
                    <ChannelCard key={ch.id} channel={ch} />
                  ))}
                </div>
              </div>
            ) : (
              <EmptyState
                icon={RefreshCw}
                title="Brak kanałów synchronizacji"
                description="Dodaj kanał sprzedaży, aby rozpocząć synchronizację stanów magazynowych"
              />
            )}

            {/* Link to events log */}
            <div className="flex justify-end">
              <Link href="/stock-sync/events">
                <Button variant="outline">
                  Historia zdarzeń
                </Button>
              </Link>
            </div>
          </>
        )}
      </div>
    </AdminGuard>
  );
}
