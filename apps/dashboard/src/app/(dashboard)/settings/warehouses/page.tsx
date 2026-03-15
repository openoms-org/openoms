"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Warehouse, Trash2, Plus } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useWarehouses, useDeleteWarehouse, useCreateWarehouse } from "@/hooks/use-warehouses";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useTranslations } from "next-intl";

export default function WarehousesPage() {
  const t = useTranslations("warehouses");
  const tc = useTranslations("common");
  const router = useRouter();
  const { data, isLoading, isError, refetch } = useWarehouses();
  const deleteWarehouse = useDeleteWarehouse();
  const createWarehouse = useCreateWarehouse();

  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newCode, setNewCode] = useState("");

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const warehouses = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deleteWarehouse.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t("deleted"));
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleCreate = () => {
    if (!newName.trim()) return;
    createWarehouse.mutate(
      { name: newName, code: newCode || undefined },
      {
        onSuccess: () => {
          toast.success(t("created"));
          setShowCreate(false);
          setNewName("");
          setNewCode("");
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      }
    );
  };

  return (
    <AdminGuard>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("title")}</h1>
          <p className="text-muted-foreground">
            {t("subtitle")}
          </p>
        </div>
        <Dialog open={showCreate} onOpenChange={setShowCreate}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              {t("newWarehouse")}
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("newWarehouse")}</DialogTitle>
              <DialogDescription>
                {t("addDescription")}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="new-name">{t("columns.name")}</Label>
                <Input
                  id="new-name"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder={t("namePlaceholder")}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-code">{t("columns.code")}</Label>
                <Input
                  id="new-code"
                  value={newCode}
                  onChange={(e) => setNewCode(e.target.value)}
                  placeholder={t("codePlaceholder")}
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setShowCreate(false)}
              >
                {t("cancel")}
              </Button>
              <Button
                onClick={handleCreate}
                disabled={!newName.trim() || createWarehouse.isPending}
              >
                {createWarehouse.isPending ? t("creating") : tc("create")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            {t("loadError")}
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

      {warehouses.length === 0 ? (
        <EmptyState
          icon={Warehouse}
          title={t("empty.title")}
          description={t("empty.description")}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("columns.name")}</TableHead>
                <TableHead>{t("columns.code")}</TableHead>
                <TableHead>{t("columns.default")}</TableHead>
                <TableHead>{t("columns.active")}</TableHead>
                <TableHead>{t("columns.createdAt")}</TableHead>
                <TableHead className="w-[80px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {warehouses.map((warehouse) => (
                <TableRow
                  key={warehouse.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() =>
                    router.push(`/settings/warehouses/${warehouse.id}`)
                  }
                >
                  <TableCell className="font-medium">
                    {warehouse.name}
                  </TableCell>
                  <TableCell>{warehouse.code || "---"}</TableCell>
                  <TableCell>
                    {warehouse.is_default ? (
                      <Badge variant="default">{t("yes")}</Badge>
                    ) : (
                      <span className="text-muted-foreground">{t("no")}</span>
                    )}
                  </TableCell>
                  <TableCell>
                    {warehouse.active ? (
                      <Badge
                        variant="outline"
                        className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                      >
                        {t("columns.active")}
                      </Badge>
                    ) : (
                      <Badge
                        variant="outline"
                        className="bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
                      >
                        {t("inactive")}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>{formatDate(warehouse.created_at)}</TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleteId(warehouse.id);
                      }}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t("deleteConfirm.title")}
        description={t("deleteConfirm.description")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteWarehouse.isPending}
      />
    </AdminGuard>
  );
}
