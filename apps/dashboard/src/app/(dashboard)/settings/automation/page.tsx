"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useAutomationRules, useDeleteAutomationRule } from "@/hooks/use-automation";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import {
  AUTOMATION_TRIGGER_EVENTS,
  AUTOMATION_TRIGGER_LABELS,
} from "@/lib/constants";
import { Plus, Trash2, Loader2 } from "lucide-react";
import type { AutomationRule } from "@/types/api";

export default function AutomationRulesPage() {
  const router = useRouter();
  const [triggerFilter, setTriggerFilter] = useState<string>("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);

  const { data, isLoading, isError, refetch } = useAutomationRules({
    trigger_event: triggerFilter || undefined,
    limit,
    offset,
  });

  const deleteRule = useDeleteAutomationRule();

  const handleTriggerChange = (value: string) => {
    setTriggerFilter(value === "all" ? "" : value);
    setOffset(0);
  };

  const handleDelete = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    if (!confirm("Czy na pewno chcesz usunąć tę regułę?")) return;
    try {
      await deleteRule.mutateAsync(id);
      toast.success("Reguła została usunięta");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Nie udało się usunąć reguły";
      toast.error(message);
    }
  };

  return (
    <AdminGuard>
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Automatyzacja</h1>
          <p className="text-muted-foreground mt-1">
            Reguły automatycznego przetwarzania zdarzeń
          </p>
          <p className="text-sm text-muted-foreground">
            Reguły automatyzacji wykonują wewnętrzne akcje (zmiana statusu, wysyłka powiadomień) gdy spełnione są warunki.
          </p>
        </div>
        <Button onClick={() => router.push("/settings/automation/new")}>
          <Plus className="h-4 w-4" />
          Nowa reguła
        </Button>
      </div>

      <div className="flex items-center gap-4">
        <div className="w-[280px]">
          <Select value={triggerFilter || "all"} onValueChange={handleTriggerChange}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Zdarzenie wyzwalające" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszystkie zdarzenia</SelectItem>
              {AUTOMATION_TRIGGER_EVENTS.map((event) => (
                <SelectItem key={event} value={event}>
                  {AUTOMATION_TRIGGER_LABELS[event] || event}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystąpił błąd podczas ładowania danych. Spróbuj odświeżyć stronę.
          </p>
          <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
            Spróbuj ponownie
          </Button>
        </div>
      )}

      {isLoading ? (
        <LoadingSkeleton />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nazwa</TableHead>
                <TableHead>Zdarzenie</TableHead>
                <TableHead>Priorytet</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Wykonania</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {data?.items && data.items.length > 0 ? (
                data.items.map((rule: AutomationRule) => (
                  <TableRow
                    key={rule.id}
                    className="cursor-pointer"
                    onClick={() => router.push(`/settings/automation/${rule.id}`)}
                  >
                    <TableCell>
                      <div>
                        <div className="font-medium">{rule.name}</div>
                        {rule.description && (
                          <div className="text-sm text-muted-foreground truncate max-w-[300px]">
                            {rule.description}
                          </div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {AUTOMATION_TRIGGER_LABELS[rule.trigger_event] || rule.trigger_event}
                      </Badge>
                    </TableCell>
                    <TableCell>{rule.priority}</TableCell>
                    <TableCell>
                      <Badge
                        className={
                          rule.enabled
                            ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                            : "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
                        }
                      >
                        {rule.enabled ? "Aktywna" : "Wyłączona"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">{rule.fire_count}</TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => handleDelete(e, rule.id)}
                        disabled={deleteRule.isPending}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">
                    Brak reguł automatyzacji
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      )}

      {data && (
        <DataTablePagination
          total={data.total}
          limit={limit}
          offset={offset}
          onPageChange={setOffset}
          onPageSizeChange={(newLimit) => {
            setLimit(newLimit);
            setOffset(0);
          }}
        />
      )}
    </div>
    </AdminGuard>
  );
}
