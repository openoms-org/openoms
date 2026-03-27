"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Factory, Trash2, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useSuppliers, useDeleteSupplier, useSyncSupplier } from "@/hooks/use-suppliers";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { StatusBadge } from "@/components/shared/status-badge";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { SUPPLIER_STATUSES } from "@/lib/constants";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useTranslations } from "next-intl";

export default function SuppliersPage() {
  const t = useTranslations("suppliers");
  const tc = useTranslations("common");
  const router = useRouter();
  const { data, isLoading, isError, refetch } = useSuppliers();
  const deleteSupplier = useDeleteSupplier();
  const syncSupplier = useSyncSupplier();

  const [deleteId, setDeleteId] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const suppliers = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deleteSupplier.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t("supplierDeleted"));
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleSync = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    syncSupplier.mutate(id, {
      onSuccess: () => {
        toast.success(t("synchronizacjaZakonczona"));
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <AdminGuard>
      <PageHeader
        title={t("title")}
        description={t("zarzadzajDostawcamiISynchronizacjaFeedowProduktowy")}
        action={{ label: t("newSupplier"), href: "/suppliers/new" }}
      />

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

      {suppliers.length === 0 ? (
        <EmptyState
          icon={Factory}
          title={t("brakDostawcow")}
          description={t("dodajPierwszegoDostawceAbyImportowacProduktyZFeedo")}
          action={{ label: t("newSupplier"), href: "/suppliers/new" }}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{tc("name")}</TableHead>
                <TableHead>{tc("code")}</TableHead>
                <TableHead>{t("format")}</TableHead>
                <TableHead>{tc("status")}</TableHead>
                <TableHead>{t("lastSync")}</TableHead>
                <TableHead>{tc("createdAt")}</TableHead>
                <TableHead className="w-[100px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {suppliers.map((supplier) => (
                <TableRow
                  key={supplier.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() => router.push(`/suppliers/${supplier.id}`)}
                >
                  <TableCell className="font-medium">{supplier.name}</TableCell>
                  <TableCell>{supplier.code || "---"}</TableCell>
                  <TableCell className="uppercase">{supplier.feed_format}</TableCell>
                  <TableCell>
                    <StatusBadge
                      status={supplier.status}
                      statusMap={SUPPLIER_STATUSES}
                    />
                  </TableCell>
                  <TableCell>
                    {supplier.last_sync_at
                      ? formatDate(supplier.last_sync_at)
                      : "---"}
                  </TableCell>
                  <TableCell>{formatDate(supplier.created_at)}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        onClick={(e) => handleSync(supplier.id, e)}
                        disabled={syncSupplier.isPending}
                      >
                        <RefreshCw className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteId(supplier.id);
                        }}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
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
        title={t("usunDostawce")}
        description={t("czyNaPewnoChceszUsunacTegoDostawceProduktyDostawcy")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteSupplier.isPending}
      />
    </AdminGuard>
  );
}
