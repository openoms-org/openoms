"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ClipboardList, Trash2, Plus } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useWarehouseDocuments,
  useDeleteWarehouseDocument,
} from "@/hooks/use-warehouse-documents";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
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
import Link from "next/link";
import { useTranslations } from "next-intl";

const STATUS_COLORS: Record<string, string> = {
  draft: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  confirmed: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  cancelled: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200",
};

export default function WarehouseDocumentsPage() {
  const t = useTranslations("warehouseDocuments");
  const tc = useTranslations("common");
  const router = useRouter();
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const { data, isLoading, isError, refetch } = useWarehouseDocuments({
    document_type: typeFilter === "all" ? undefined : typeFilter || undefined,
    status: statusFilter === "all" ? undefined : statusFilter || undefined,
    limit: 50,
  });
  const deleteDocument = useDeleteWarehouseDocument();

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const documents = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deleteDocument.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t("deleted"));
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <AdminGuard>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("title")}
          </h1>
          <p className="text-muted-foreground">
            {t("zarzadzajDokumentamiPzWzIMm")}
          </p>
        </div>
        <Button asChild>
          <Link href="/settings/warehouse-documents/new">
            <Plus className="h-4 w-4 mr-2" />
            {t("newDocumentAction")}
          </Link>
        </Button>
      </div>

      <div className="flex gap-4 mb-4">
        <Select value={typeFilter} onValueChange={setTypeFilter}>
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder={t("docTypeLabel")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("allTypes")}</SelectItem>
            <SelectItem value="PZ">PZ</SelectItem>
            <SelectItem value="WZ">WZ</SelectItem>
            <SelectItem value="MM">MM</SelectItem>
          </SelectContent>
        </Select>

        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder={t("columns.status")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("allStatuses")}</SelectItem>
            <SelectItem value="draft">{t("statusDraft")}</SelectItem>
            <SelectItem value="confirmed">{t("statusConfirmed")}</SelectItem>
            <SelectItem value="cancelled">{t("statusCancelled")}</SelectItem>
          </SelectContent>
        </Select>
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

      {documents.length === 0 ? (
        <EmptyState
          icon={ClipboardList}
          title={t("brakDokumentowMagazynowych")}
          description={t("utworzPierwszyDokumentPzWzLubMmAbyZarzadzacRuchemT")}
          action={{
            label: t("newDocumentAction"),
            href: "/settings/warehouse-documents/new",
          }}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("columns.number")}</TableHead>
                <TableHead>{t("columns.type")}</TableHead>
                <TableHead>{t("columns.status")}</TableHead>
                <TableHead>{t("columns.createdAt")}</TableHead>
                <TableHead>{t("columns.confirmedAt")}</TableHead>
                <TableHead className="w-[80px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {documents.map((doc) => (
                <TableRow
                  key={doc.id}
                  className="cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() =>
                    router.push(`/settings/warehouse-documents/${doc.id}`)
                  }
                >
                  <TableCell className="font-medium">
                    {doc.document_number}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{doc.document_type}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant="outline"
                      className={STATUS_COLORS[doc.status] || ""}
                    >
                      {t(`status${doc.status.charAt(0).toUpperCase()}${doc.status.slice(1)}`, { defaultValue: doc.status })}
                    </Badge>
                  </TableCell>
                  <TableCell>{formatDate(doc.created_at)}</TableCell>
                  <TableCell>
                    {doc.confirmed_at
                      ? formatDate(doc.confirmed_at)
                      : "---"}
                  </TableCell>
                  <TableCell>
                    {doc.status === "draft" && (
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteId(doc.id);
                        }}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    )}
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
        title={t("usunDokument")}
        description={t("czyNaPewnoChceszUsunacTenDokumentMagazynowy")}
        confirmLabel={tc("delete")}
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteDocument.isPending}
      />
    </AdminGuard>
  );
}
