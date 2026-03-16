"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useAutomationRules, useDeleteAutomationRule } from "@/hooks/use-automation";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import {
  AUTOMATION_TRIGGER_LABELS,
} from "@/lib/constants";
import { Plus, Trash2, Zap, GitBranch, Pencil } from "lucide-react";
import type { AutomationRule } from "@/types/api";
import { useTranslations } from "next-intl";

export default function WorkflowsPage() {
  const t = useTranslations("workflows");
  const router = useRouter();
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const { data, isLoading, isError, refetch } = useAutomationRules({ limit, offset });
  const deleteRule = useDeleteAutomationRule();

  const handleDeleteClick = (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    setDeleteId(id);
  };

  const handleDeleteConfirm = async () => {
    if (!deleteId) return;
    try {
      await deleteRule.mutateAsync(deleteId);
      toast.success(t("deleteSuccess"));
    } catch (err) {
      const message = err instanceof Error ? err.message : t("deleteError");
      toast.error(message);
    } finally {
      setDeleteId(null);
    }
  };

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">{t("title")}</h1>
            <p className="text-muted-foreground mt-1">
              {t("description")}
            </p>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => router.push("/settings/automation")}
            >
              <GitBranch className="h-4 w-4" />
              {t("textEditor")}
            </Button>
            <Button onClick={() => router.push("/workflows/new")}>
              <Plus className="h-4 w-4" />
              {t("newWorkflow")}
            </Button>
          </div>
        </div>

        {isError && (
          <div className="rounded-md border border-destructive bg-destructive/10 p-4">
            <p className="text-sm text-destructive">
              {t("errorLoading")}
            </p>
            <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
              {t("retry")}
            </Button>
          </div>
        )}

        {isLoading ? (
          <LoadingSkeleton />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data?.items && data.items.length > 0 ? (
              data.items.map((rule: AutomationRule) => (
                <Card
                  key={rule.id}
                  className="cursor-pointer hover:shadow-md transition-shadow group"
                  onClick={() => router.push(`/workflows/${rule.id}`)}
                >
                  <CardHeader className="pb-3">
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-2">
                        <div className="p-1.5 rounded bg-green-500">
                          <Zap className="h-3.5 w-3.5 text-white" />
                        </div>
                        <CardTitle className="text-base">{rule.name}</CardTitle>
                      </div>
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation();
                            router.push(`/workflows/${rule.id}`);
                          }}
                        >
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => handleDeleteClick(e, rule.id)}
                          disabled={deleteRule.isPending}
                        >
                          <Trash2 className="h-3.5 w-3.5 text-destructive" />
                        </Button>
                      </div>
                    </div>
                    {rule.description && (
                      <CardDescription className="truncate">{rule.description}</CardDescription>
                    )}
                  </CardHeader>
                  <CardContent>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className="text-xs">
                          {AUTOMATION_TRIGGER_LABELS[rule.trigger_event] || rule.trigger_event}
                        </Badge>
                        <Badge
                          className={`text-xs ${
                            rule.enabled
                              ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                              : "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
                          }`}
                        >
                          {rule.enabled ? t("statusActive") : t("statusDraft")}
                        </Badge>
                      </div>
                      <span className="text-xs text-muted-foreground">
                        {t("executions", { count: rule.fire_count })}
                      </span>
                    </div>
                  </CardContent>
                </Card>
              ))
            ) : (
              <div className="col-span-full flex flex-col items-center justify-center py-12 text-center">
                <Zap className="h-12 w-12 text-muted-foreground/50 mb-4" />
                <h3 className="text-lg font-medium">{t("emptyTitle")}</h3>
                <p className="text-sm text-muted-foreground mt-1">
                  {t("emptyDescription")}
                </p>
                <Button className="mt-4" onClick={() => router.push("/workflows/new")}>
                  <Plus className="h-4 w-4" />
                  {t("newWorkflow")}
                </Button>
              </div>
            )}
          </div>
        )}

        {data && data.total > limit && (
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

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t("deleteConfirm")}
        description={t("deleteConfirmDescription")}
        variant="destructive"
        onConfirm={handleDeleteConfirm}
        isLoading={deleteRule.isPending}
      />
    </AdminGuard>
  );
}
