"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { TrendingUp, Pause, Play, Zap, Activity, BarChart3, Hash } from "lucide-react";
import {
  useRepricingRules,
  useRepricingSummary,
  useUpdateRepricingRule,
  useApplyRepricingRules,
} from "@/hooks/use-repricing";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { StatusBadge } from "@/components/shared/status-badge";
import { EmptyState } from "@/components/shared/empty-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { formatDate } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import type { RepricingRule } from "@/types/api";
import { useTranslations } from "next-intl";

export default function RepricingPage() {
  const t = useTranslations("repricing");

  const RULE_STATUSES: Record<string, { label: string; color: string }> = {
    active: {
      label: t("statusActive"),
      color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
    },
    paused: {
      label: t("statusPaused"),
      color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
    },
    archived: {
      label: t("statusArchived"),
      color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200",
    },
  };

  const STRATEGY_LABELS: Record<string, string> = {
    margin: t("strategyMargin"),
    competitive: t("strategyCompetitive"),
    time_based: t("strategyTimeBased"),
    stock_based: t("strategyStockBased"),
  };

  const SCOPE_LABELS: Record<string, string> = {
    all: t("scopeAll"),
    category: t("scopeCategory"),
    tag: t("scopeTag"),
    product_ids: t("scopeProductIds"),
  };
  const router = useRouter();
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);

  const { data, isLoading, isError, refetch } = useRepricingRules({
    status: statusFilter || undefined,
    limit,
    offset,
  });

  const { data: summary } = useRepricingSummary();
  const updateRule = useUpdateRepricingRule("");
  const applyRules = useApplyRepricingRules();

  const handleToggleStatus = async (rule: RepricingRule) => {
    const newStatus = rule.status === "active" ? "paused" : "active";
    try {
      await updateRule.mutateAsync({ status: newStatus });
      toast.success(
        newStatus === "active"
          ? t("regułaaktywowana")
          : t("reguławstrzymana")
      );
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleApplyNow = async () => {
    try {
      const result = await applyRules.mutateAsync();
      toast.success(
        t("repricingCompleted", { priceChanges: result.price_changes, productsAffected: result.products_affected })
      );
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const columns: ColumnDef<RepricingRule>[] = [
    {
      header: t("name"),
      accessorKey: "name",
      cell: (row) => <span className="font-medium">{row.name}</span>,
    },
    {
      header: t("strategy"),
      accessorKey: "strategy",
      cell: (row) => (
        <span className="text-sm">
          {STRATEGY_LABELS[row.strategy] || row.strategy}
        </span>
      ),
    },
    {
      header: t("scope"),
      accessorKey: "scope_type",
      cell: (row) => (
        <span className="text-sm text-muted-foreground">
          {SCOPE_LABELS[row.scope_type] || row.scope_type}
        </span>
      ),
    },
    {
      header: t("status"),
      accessorKey: "status",
      cell: (row) => (
        <StatusBadge status={row.status} statusMap={RULE_STATUSES} />
      ),
    },
    {
      header: t("priority"),
      accessorKey: "priority",
      cell: (row) => <span className="text-sm font-mono">{row.priority}</span>,
    },
    {
      header: t("products"),
      accessorKey: "products_affected",
      cell: (row) => <span className="text-sm">{row.products_affected}</span>,
    },
    {
      header: t("ostatniwpływ"),
      accessorKey: "last_applied_at",
      cell: (row) =>
        row.last_applied_at ? (
          <span className="text-sm text-muted-foreground">
            {formatDate(row.last_applied_at)}
          </span>
        ) : (
          <span className="text-sm text-muted-foreground">-</span>
        ),
    },
    {
      header: "",
      accessorKey: "id",
      cell: (row) => (
        <Button
          variant="ghost"
          size="sm"
          onClick={(e) => {
            e.stopPropagation();
            handleToggleStatus(row);
          }}
        >
          {row.status === "active" ? (
            <Pause className="h-4 w-4" />
          ) : (
            <Play className="h-4 w-4" />
          )}
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Repricing</h1>
          <p className="text-muted-foreground mt-1">
            {t("dynamiczneZarzadzanieCenamiProduktow")}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={handleApplyNow}
            disabled={applyRules.isPending}
          >
            <Zap className="mr-2 h-4 w-4" />
            {applyRules.isPending ? t("applying") : t("applyNow")}
          </Button>
          <Button asChild>
            <Link href="/repricing/new">{t("newRule")}</Link>
          </Button>
        </div>
      </div>

      {/* Summary cards */}
      {summary && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-2">
                <Activity className="h-4 w-4 text-green-500" />
                <span className="text-sm text-muted-foreground">
                  {t("aktywneReguły")}
                </span>
              </div>
              <p className="mt-1 text-2xl font-bold">{summary.active_rules}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-2">
                <TrendingUp className="h-4 w-4 text-blue-500" />
                <span className="text-sm text-muted-foreground">
                  {t("changesToday")}
                </span>
              </div>
              <p className="mt-1 text-2xl font-bold">
                {summary.changes_today}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-2">
                <BarChart3 className="h-4 w-4 text-purple-500" />
                <span className="text-sm text-muted-foreground">
                  {t("sredniaZmiana")}
                </span>
              </div>
              <p className="mt-1 text-2xl font-bold">
                {summary.avg_change_pct.toFixed(1)}%
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center gap-2">
                <Hash className="h-4 w-4 text-orange-500" />
                <span className="text-sm text-muted-foreground">
                  {t("changesWeek")}
                </span>
              </div>
              <p className="mt-1 text-2xl font-bold">
                {summary.changes_week}
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      <div className="flex items-center gap-4">
        <div className="w-[200px]">
          <Select
            value={statusFilter || "all"}
            onValueChange={(v) => {
              setStatusFilter(v === "all" ? "" : v);
              setOffset(0);
            }}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("filterAll")}</SelectItem>
              <SelectItem value="active">{t("filterActive")}</SelectItem>
              <SelectItem value="paused">{t("filterPaused")}</SelectItem>
              <SelectItem value="archived">{t("filterArchived")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            {t("wystapiłBładPodczasŁadowaniaDanych")}
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            {t("retry")}
          </Button>
        </div>
      )}

      <div className="rounded-md border">
        <DataTable<RepricingRule>
          columns={columns}
          data={data?.items || []}
          isLoading={isLoading}
          emptyState={
            <EmptyState
              icon={TrendingUp}
              title={t("brakregułrepricing")}
              description={t("utworzpierwszaregułedynamicznegoustalaniacen")}
              action={{ label: t("newRule"), href: "/repricing/new" }}
            />
          }
          onRowClick={(row) => router.push(`/repricing/${row.id}`)}
        />
      </div>

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
  );
}
