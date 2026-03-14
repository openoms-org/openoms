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
import { Plus, Trash2 } from "lucide-react";
import type { AutomationRule } from "@/types/api";
import { useTranslations } from "next-intl";

export default function AutomationRulesPage() {
  const t = useTranslations("automation");
  const tc = useTranslations("common");
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
    if (!confirm(t("czynapewnochceszusunacteregułe"))) return;
    try {
      await deleteRule.mutateAsync(id);
      toast.success(t("regułazostałausunieta"));
    } catch (err) {
      const message = err instanceof Error ? err.message : t("nieudałosieusunacreguły");
      toast.error(message);
    }
  };

  return (
    <AdminGuard>
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("pageTitle")}</h1>
          <p className="text-muted-foreground mt-1">
            {t("regułyAutomatycznegoPrzetwarzaniaZdarzen")}
          </p>
          <p className="text-sm text-muted-foreground">
            {t("pageDescription")}
          </p>
        </div>
        <Button onClick={() => router.push("/settings/automation/new")}>
          <Plus className="h-4 w-4" />
          {t("newRule")}
        </Button>
      </div>

      <div className="flex items-center gap-4">
        <div className="w-[280px]">
          <Select value={triggerFilter || "all"} onValueChange={handleTriggerChange}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder={t("zdarzenieWyzwalajace")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("allEvents")}</SelectItem>
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
            {t("loadError")}
          </p>
          <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
            {t("retry")}
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
                <TableHead>{t("columns.name")}</TableHead>
                <TableHead>{t("columns.event")}</TableHead>
                <TableHead>{t("columns.priority")}</TableHead>
                <TableHead>{tc("status")}</TableHead>
                <TableHead className="text-right">{t("columns.executions")}</TableHead>
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
                        {rule.enabled ? t("statusActive") : t("wyłaczona1")}
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
                    {t("brakRegułAutomatyzacji")}
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
