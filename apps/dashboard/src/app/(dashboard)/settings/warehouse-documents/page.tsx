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
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
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

const DOC_TYPE_LABELS: Record<string, string> = {
  PZ: "PZ - Przyjecie zewnetrzne",
  WZ: "WZ - Wydanie zewnetrzne",
  MM: "MM - Przesuniecie miedzymagazynowe",
};

const STATUS_LABELS: Record<string, string> = {
  draft: "Szkic",
  confirmed: "Zatwierdzony",
  cancelled: "Anulowany",
};

const STATUS_COLORS: Record<string, string> = {
  draft: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  confirmed: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  cancelled: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200",
};

export default function WarehouseDocumentsPage() {
  const router = useRouter();
  const [typeFilter, setTypeFilter] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const { data, isLoading, isError, refetch } = useWarehouseDocuments({
    document_type: typeFilter || undefined,
    status: statusFilter || undefined,
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
        toast.success("Dokument zostal usuniety");
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
            Dokumenty magazynowe
          </h1>
          <p className="text-muted-foreground">
            Zarzadzaj dokumentami PZ, WZ i MM
          </p>
        </div>
        <Button asChild>
          <Link href="/settings/warehouse-documents/new">
            <Plus className="h-4 w-4 mr-2" />
            Nowy dokument
          </Link>
        </Button>
      </div>

      <div className="flex gap-4 mb-4">
        <Select value={typeFilter} onValueChange={setTypeFilter}>
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder="Typ dokumentu" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie typy</SelectItem>
            <SelectItem value="PZ">PZ</SelectItem>
            <SelectItem value="WZ">WZ</SelectItem>
            <SelectItem value="MM">MM</SelectItem>
          </SelectContent>
        </Select>

        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie statusy</SelectItem>
            <SelectItem value="draft">Szkic</SelectItem>
            <SelectItem value="confirmed">Zatwierdzony</SelectItem>
            <SelectItem value="cancelled">Anulowany</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystapil blad podczas ladowania danych. Sprobuj odswiezyc strone.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Sprobuj ponownie
          </Button>
        </div>
      )}

      {documents.length === 0 ? (
        <EmptyState
          icon={ClipboardList}
          title="Brak dokumentow magazynowych"
          description="Utworz pierwszy dokument PZ, WZ lub MM, aby zarzadzac ruchem towarow."
          action={{
            label: "Nowy dokument",
            href: "/settings/warehouse-documents/new",
          }}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Numer</TableHead>
                <TableHead>Typ</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Utworzono</TableHead>
                <TableHead>Zatwierdzono</TableHead>
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
                      {STATUS_LABELS[doc.status] || doc.status}
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
        title="Usun dokument"
        description="Czy na pewno chcesz usunac ten dokument magazynowy?"
        confirmLabel="Usun"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteDocument.isPending}
      />
    </AdminGuard>
  );
}
