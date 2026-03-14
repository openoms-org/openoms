"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { ArrowLeft, Play, Pause, Trash2, Zap, Eye } from "lucide-react";
import {
  useRepricingRule,
  useUpdateRepricingRule,
  useDeleteRepricingRule,
  useSimulateRepricingRule,
  useApplyRepricingRules,
  useRepricingLog,
} from "@/hooks/use-repricing";
import { StatusBadge } from "@/components/shared/status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { formatDate, formatCurrency } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import type { SimulationResult } from "@/types/api";
import { useTranslations } from "next-intl";

export default function RepricingRuleDetailPage() {
  const t = useTranslations("repricing");
  const tc = useTranslations("common");

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
    competitive: t("strategyCompetitiveSoon"),
    time_based: t("strategyTimeBased"),
    stock_based: t("strategyStockBased"),
  };

  const SCOPE_LABELS: Record<string, string> = {
    all: t("scopeAll"),
    category: t("scopeCategory"),
    tag: t("scopeTag"),
    product_ids: t("scopeProductIds"),
  };
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [simulationResults, setSimulationResults] = useState<
    SimulationResult[] | null
  >(null);
  const [showSimulation, setShowSimulation] = useState(false);

  const { data: rule, isLoading, refetch } = useRepricingRule(params.id);
  const updateRule = useUpdateRepricingRule(params.id);
  const deleteRule = useDeleteRepricingRule();
  const simulateRule = useSimulateRepricingRule(params.id);
  const applyRules = useApplyRepricingRules();
  const { data: logData } = useRepricingLog({
    rule_id: params.id,
    limit: 20,
    offset: 0,
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    );
  }

  if (!rule) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">{t("regułanieznaleziona")}</p>
        <Button asChild className="mt-4">
          <Link href="/repricing">{t("powrotDoListy")}</Link>
        </Button>
      </div>
    );
  }

  const handleToggleStatus = async () => {
    const newStatus = rule.status === "active" ? "paused" : "active";
    try {
      await updateRule.mutateAsync({ status: newStatus });
      toast.success(
        newStatus === "active" ? t("regułaAktywowana") : t("reguławstrzymana")
      );
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleDelete = async () => {
    try {
      await deleteRule.mutateAsync(params.id);
      toast.success(t("regułausunieta"));
      router.push("/repricing");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleSimulate = async () => {
    try {
      const results = await simulateRule.mutateAsync();
      setSimulationResults(results);
      setShowSimulation(true);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleApplyNow = async () => {
    try {
      const result = await applyRules.mutateAsync();
      toast.success(
        t("repricingCompletedShort", { priceChanges: result.price_changes })
      );
      refetch();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const ruleParams = rule.params as Record<string, unknown>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/repricing">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">{rule.name}</h1>
              <StatusBadge status={rule.status} statusMap={RULE_STATUSES} />
            </div>
            <p className="text-muted-foreground mt-1">
              {t("strategy")}: {STRATEGY_LABELS[rule.strategy] || rule.strategy} |
              {t("priority")}: {rule.priority}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleSimulate}
            disabled={simulateRule.isPending}
          >
            <Eye className="mr-2 h-4 w-4" />
            {simulateRule.isPending ? t("simulating") : t("simulate")}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleApplyNow}
            disabled={applyRules.isPending}
          >
            <Zap className="mr-2 h-4 w-4" />
            {t("applyNow")}
          </Button>
          <Button variant="outline" size="sm" onClick={handleToggleStatus}>
            {rule.status === "active" ? (
              <>
                <Pause className="mr-2 h-4 w-4" />
                {t("pause")}
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" />
                {t("activate")}
              </>
            )}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setShowDeleteDialog(true)}
          >
            <Trash2 className="mr-2 h-4 w-4" />
            {t("delete")}
          </Button>
        </div>
      </div>

      {/* Rule config */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("configuration")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t("strategy")}</span>
              <span>{STRATEGY_LABELS[rule.strategy] || rule.strategy}</span>
            </div>
            <Separator />
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t("scope")}</span>
              <span>{SCOPE_LABELS[rule.scope_type] || rule.scope_type}</span>
            </div>
            {rule.scope_value != null && (
              <>
                <Separator />
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">{t("wartoscZakresu")}</span>
                  <span className="max-w-[200px] truncate text-right">
                    {typeof rule.scope_value === "string"
                      ? rule.scope_value
                      : JSON.stringify(rule.scope_value)}
                  </span>
                </div>
              </>
            )}
            <Separator />
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t("priority")}</span>
              <span>{rule.priority}</span>
            </div>
            {rule.min_price != null && (
              <>
                <Separator />
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">{t("minPrice")}</span>
                  <span>{formatCurrency(rule.min_price)}</span>
                </div>
              </>
            )}
            {rule.max_price != null && (
              <>
                <Separator />
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">{t("maxPrice")}</span>
                  <span>{formatCurrency(rule.max_price)}</span>
                </div>
              </>
            )}
            <Separator />
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t("products")}</span>
              <span>{rule.products_affected}</span>
            </div>
            {rule.last_applied_at && (
              <>
                <Separator />
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">
                    {t("lastApplied")}
                  </span>
                  <span>{formatDate(rule.last_applied_at)}</span>
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("strategyParams")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {Object.entries(ruleParams).map(([key, value]) => (
              <div key={key}>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">{key}</span>
                  <span className="max-w-[200px] truncate text-right">
                    {typeof value === "object"
                      ? JSON.stringify(value)
                      : String(value)}
                  </span>
                </div>
                <Separator className="mt-3" />
              </div>
            ))}
            {Object.keys(ruleParams).length === 0 && (
              <p className="text-sm text-muted-foreground">{t("brakParametrow")}</p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Recent price changes */}
      <Card>
        <CardHeader>
          <CardTitle>{t("recentPriceChanges")}</CardTitle>
        </CardHeader>
        <CardContent>
          {logData && logData.items.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("product")}</TableHead>
                  <TableHead>{t("oldPrice")}</TableHead>
                  <TableHead>{t("newPrice")}</TableHead>
                  <TableHead>{t("change")}</TableHead>
                  <TableHead>{t("detail.reason")}</TableHead>
                  <TableHead>{t("date")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logData.items.map((log) => {
                  const changePct =
                    log.old_price > 0
                      ? (
                          ((log.new_price - log.old_price) / log.old_price) *
                          100
                        ).toFixed(1)
                      : "0";
                  const isIncrease = log.new_price > log.old_price;
                  return (
                    <TableRow key={log.id}>
                      <TableCell className="font-mono text-xs">
                        <Link
                          href={`/products/${log.product_id}`}
                          className="text-primary hover:underline"
                        >
                          {log.product_id.slice(0, 8)}...
                        </Link>
                      </TableCell>
                      <TableCell>{formatCurrency(log.old_price)}</TableCell>
                      <TableCell>{formatCurrency(log.new_price)}</TableCell>
                      <TableCell>
                        <span
                          className={
                            isIncrease ? "text-green-600" : "text-red-600"
                          }
                        >
                          {isIncrease ? "+" : ""}
                          {changePct}%
                        </span>
                      </TableCell>
                      <TableCell className="max-w-[200px] truncate text-sm text-muted-foreground">
                        {log.reason || "-"}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatDate(log.applied_at)}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          ) : (
            <p className="text-sm text-muted-foreground py-4 text-center">
              {t("brakZmianCenDlaTejReguły")}
            </p>
          )}
        </CardContent>
      </Card>

      {/* Simulation dialog */}
      <Dialog open={showSimulation} onOpenChange={setShowSimulation}>
        <DialogContent className="max-w-3xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{t("simulationResults")}</DialogTitle>
          </DialogHeader>
          {simulationResults && simulationResults.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("product")}</TableHead>
                  <TableHead>{t("sku")}</TableHead>
                  <TableHead>{t("oldPrice")}</TableHead>
                  <TableHead>{t("newPrice")}</TableHead>
                  <TableHead>{t("change")}</TableHead>
                  <TableHead>{t("detail.reason")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {simulationResults.map((result) => (
                  <TableRow key={result.product_id}>
                    <TableCell className="font-medium max-w-[150px] truncate">
                      {result.product_name}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {result.sku || "-"}
                    </TableCell>
                    <TableCell>{formatCurrency(result.old_price)}</TableCell>
                    <TableCell>{formatCurrency(result.new_price)}</TableCell>
                    <TableCell>
                      <span
                        className={
                          result.change_pct >= 0
                            ? "text-green-600"
                            : "text-red-600"
                        }
                      >
                        {result.change_pct >= 0 ? "+" : ""}
                        {result.change_pct.toFixed(1)}%
                      </span>
                    </TableCell>
                    <TableCell className="max-w-[200px] truncate text-sm text-muted-foreground">
                      {result.reason}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <p className="text-sm text-muted-foreground py-8 text-center">
              {rule.strategy === "competitive"
                ? t("strategiaKonkurencyjnaJestWPrzygotowaniuDaneOCenac")
                : t("brakZmianDoZastosowaniaWszystkieCenySaAktualne")}
            </p>
          )}
        </DialogContent>
      </Dialog>

      {/* Delete dialog */}
      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title={t("usunacregułe")}
        description={t("deleteRuleConfirmation", { name: rule.name })}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  );
}
