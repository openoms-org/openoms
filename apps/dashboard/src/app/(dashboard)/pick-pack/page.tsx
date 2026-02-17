"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { PackageCheck, Plus, Play } from "lucide-react";
import { toast } from "sonner";
import { usePickPackSessions, useCreatePickPackSession } from "@/hooks/use-pick-pack";
import { useOrders } from "@/hooks/use-orders";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { PageHeader } from "@/components/shared/page-header";
import { formatDate } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
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
import type { PickPackSession, Order } from "@/types/api";

const statusLabels: Record<string, string> = {
  picking: "Kompletowanie",
  packing: "Pakowanie",
  completed: "Zakonczone",
  cancelled: "Anulowane",
};

const statusVariants: Record<string, string> = {
  picking: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  packing: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  completed: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  cancelled: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
};

function SessionStatusBadge({ status }: { status: string }) {
  return (
    <Badge variant="outline" className={statusVariants[status] || ""}>
      {statusLabels[status] || status}
    </Badge>
  );
}

export default function PickPackPage() {
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedOrderIds, setSelectedOrderIds] = useState<Set<string>>(new Set());

  const { data, isLoading, isError, refetch } = usePickPackSessions({
    status: statusFilter && statusFilter !== "all" ? statusFilter : undefined,
    limit: 50,
  });

  const { data: ordersData, isLoading: ordersLoading } = useOrders({
    status: "processing",
    limit: 100,
    sort_by: "created_at",
    sort_order: "asc",
  });

  const createSession = useCreatePickPackSession();

  const processingOrders: Order[] = ordersData?.items ?? [];
  const sessions = data?.items ?? [];

  const handleToggleOrder = (orderId: string) => {
    setSelectedOrderIds((prev) => {
      const next = new Set(prev);
      if (next.has(orderId)) {
        next.delete(orderId);
      } else {
        next.add(orderId);
      }
      return next;
    });
  };

  const handleCreateSession = async () => {
    if (selectedOrderIds.size === 0) return;
    try {
      const session = await createSession.mutateAsync({
        order_ids: Array.from(selectedOrderIds),
      });
      toast.success("Sesja Pick & Pack utworzona");
      setDialogOpen(false);
      setSelectedOrderIds(new Set());
      router.push(`/pick-pack/${session.id}`);
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  const handlePickNext = async () => {
    if (processingOrders.length === 0) {
      toast.info("Brak zamowien do kompletacji");
      return;
    }
    try {
      const session = await createSession.mutateAsync({
        order_ids: [processingOrders[0].id],
      });
      toast.success("Sesja Pick & Pack utworzona");
      router.push(`/pick-pack/${session.id}`);
    } catch (err) {
      toast.error(getErrorMessage(err));
    }
  };

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  return (
    <div>
      <PageHeader
        title="Pick & Pack"
        description="Kompletowanie i pakowanie zamowien"
      />

      <div className="flex flex-wrap gap-3 mb-6">
        <Button size="lg" onClick={handlePickNext} disabled={processingOrders.length === 0}>
          <Play className="h-5 w-5 mr-2" />
          Kompletuj nastepne
          {processingOrders.length > 0 && (
            <Badge variant="secondary" className="ml-2">
              {processingOrders.length}
            </Badge>
          )}
        </Button>
        <Button variant="outline" size="lg" onClick={() => setDialogOpen(true)}>
          <Plus className="h-5 w-5 mr-2" />
          Nowa sesja
        </Button>
      </div>

      <div className="flex gap-4 mb-4">
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder="Wszystkie statusy" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie statusy</SelectItem>
            <SelectItem value="picking">Kompletowanie</SelectItem>
            <SelectItem value="packing">Pakowanie</SelectItem>
            <SelectItem value="completed">Zakonczone</SelectItem>
            <SelectItem value="cancelled">Anulowane</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4 mb-4">
          <p className="text-sm text-destructive">
            Blad podczas ladowania danych.
          </p>
          <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
            Sprobuj ponownie
          </Button>
        </div>
      )}

      {sessions.length === 0 ? (
        <EmptyState
          icon={PackageCheck}
          title="Brak sesji Pick & Pack"
          description="Utworz nowa sesje, aby rozpoczac kompletowanie zamowien."
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Typ</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Zamowienia</TableHead>
                <TableHead>Postep kompletacji</TableHead>
                <TableHead>Postep pakowania</TableHead>
                <TableHead>Rozpoczeto</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sessions.map((session: PickPackSession) => (
                <TableRow
                  key={session.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() => router.push(`/pick-pack/${session.id}`)}
                >
                  <TableCell className="font-medium">
                    {session.session_type === "batch" ? "Wsadowa" : "Pojedyncza"}
                  </TableCell>
                  <TableCell>
                    <SessionStatusBadge status={session.status} />
                  </TableCell>
                  <TableCell>{session.stats?.order_count ?? "---"}</TableCell>
                  <TableCell>
                    {session.stats ? (
                      <div className="flex items-center gap-2">
                        <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden max-w-24">
                          <div
                            className="h-full bg-blue-500 rounded-full transition-all"
                            style={{
                              width: `${session.stats.total_required > 0 ? (session.stats.total_picked / session.stats.total_required) * 100 : 0}%`,
                            }}
                          />
                        </div>
                        <span className="text-sm text-muted-foreground whitespace-nowrap">
                          {session.stats.total_picked}/{session.stats.total_required}
                        </span>
                      </div>
                    ) : (
                      "---"
                    )}
                  </TableCell>
                  <TableCell>
                    {session.stats ? (
                      <div className="flex items-center gap-2">
                        <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden max-w-24">
                          <div
                            className="h-full bg-green-500 rounded-full transition-all"
                            style={{
                              width: `${session.stats.total_required > 0 ? (session.stats.total_packed / session.stats.total_required) * 100 : 0}%`,
                            }}
                          />
                        </div>
                        <span className="text-sm text-muted-foreground whitespace-nowrap">
                          {session.stats.total_packed}/{session.stats.total_required}
                        </span>
                      </div>
                    ) : (
                      "---"
                    )}
                  </TableCell>
                  <TableCell>{formatDate(session.started_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {/* New Session Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg max-h-[80vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle>Nowa sesja Pick & Pack</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground mb-4">
            Wybierz zamowienia do kompletacji (status: przetwarzanie)
          </p>
          <div className="flex-1 overflow-y-auto border rounded-md">
            {ordersLoading ? (
              <div className="p-4">
                <LoadingSkeleton />
              </div>
            ) : processingOrders.length === 0 ? (
              <div className="p-8 text-center text-sm text-muted-foreground">
                Brak zamowien w statusie &quot;przetwarzanie&quot;
              </div>
            ) : (
              <div className="divide-y">
                {processingOrders.map((order) => (
                  <label
                    key={order.id}
                    className="flex items-center gap-3 p-3 hover:bg-muted/50 cursor-pointer transition-colors"
                  >
                    <Checkbox
                      checked={selectedOrderIds.has(order.id)}
                      onCheckedChange={() => handleToggleOrder(order.id)}
                    />
                    <div className="flex-1 min-w-0">
                      <p className="font-medium text-sm truncate">
                        {order.customer_name}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        #{order.id.slice(0, 8)} &middot;{" "}
                        {order.items?.length ?? 0} pozycji &middot;{" "}
                        {order.total_amount} {order.currency}
                      </p>
                    </div>
                  </label>
                ))}
              </div>
            )}
          </div>
          <DialogFooter className="mt-4">
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Anuluj
            </Button>
            <Button
              onClick={handleCreateSession}
              disabled={selectedOrderIds.size === 0 || createSession.isPending}
            >
              {createSession.isPending ? "Tworzenie..." : `Utworz sesje (${selectedOrderIds.size})`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
