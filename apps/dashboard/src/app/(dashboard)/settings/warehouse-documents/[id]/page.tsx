"use client";

import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useWarehouseDocument,
  useConfirmWarehouseDocument,
  useCancelWarehouseDocument,
} from "@/hooks/use-warehouse-documents";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ArrowLeft, CheckCircle, XCircle } from "lucide-react";
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

export default function WarehouseDocumentDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;

  const { data: doc, isLoading, isError, refetch } = useWarehouseDocument(id);
  const confirmDoc = useConfirmWarehouseDocument();
  const cancelDoc = useCancelWarehouseDocument();

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError || !doc) {
    return (
      <AdminGuard>
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Nie udalo sie zaladowac dokumentu.
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
      </AdminGuard>
    );
  }

  const handleConfirm = () => {
    confirmDoc.mutate(id, {
      onSuccess: () => {
        toast.success("Dokument zostal zatwierdzony. Stany magazynowe zaktualizowane.");
        refetch();
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleCancel = () => {
    cancelDoc.mutate(id, {
      onSuccess: () => {
        toast.success("Dokument zostal anulowany");
        refetch();
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const items = doc.items ?? [];

  return (
    <AdminGuard>
      <div className="mb-6">
        <Button variant="ghost" size="sm" asChild className="mb-4">
          <Link href="/settings/warehouse-documents">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Powrot do listy
          </Link>
        </Button>

        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {doc.document_number}
            </h1>
            <p className="text-muted-foreground">
              {DOC_TYPE_LABELS[doc.document_type] || doc.document_type}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Badge
              variant="outline"
              className={`text-sm px-3 py-1 ${STATUS_COLORS[doc.status] || ""}`}
            >
              {STATUS_LABELS[doc.status] || doc.status}
            </Badge>
            {doc.status === "draft" && (
              <>
                <Button
                  onClick={handleConfirm}
                  disabled={confirmDoc.isPending}
                >
                  <CheckCircle className="h-4 w-4 mr-2" />
                  {confirmDoc.isPending ? "Zatwierdzanie..." : "Zatwierdz"}
                </Button>
                <Button
                  variant="outline"
                  onClick={handleCancel}
                  disabled={cancelDoc.isPending}
                >
                  <XCircle className="h-4 w-4 mr-2" />
                  {cancelDoc.isPending ? "Anulowanie..." : "Anuluj"}
                </Button>
              </>
            )}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-6 mb-6">
        <div className="rounded-md border p-4 space-y-2">
          <h3 className="font-semibold text-sm text-muted-foreground uppercase">
            Szczegoly
          </h3>
          <div className="grid grid-cols-2 gap-2 text-sm">
            <span className="text-muted-foreground">Typ:</span>
            <span>{doc.document_type}</span>
            <span className="text-muted-foreground">Magazyn:</span>
            <span className="font-mono text-xs">{doc.warehouse_id}</span>
            {doc.target_warehouse_id && (
              <>
                <span className="text-muted-foreground">Magazyn docelowy:</span>
                <span className="font-mono text-xs">
                  {doc.target_warehouse_id}
                </span>
              </>
            )}
            {doc.supplier_id && (
              <>
                <span className="text-muted-foreground">Dostawca:</span>
                <span className="font-mono text-xs">{doc.supplier_id}</span>
              </>
            )}
            {doc.order_id && (
              <>
                <span className="text-muted-foreground">Zamowienie:</span>
                <span className="font-mono text-xs">{doc.order_id}</span>
              </>
            )}
          </div>
        </div>

        <div className="rounded-md border p-4 space-y-2">
          <h3 className="font-semibold text-sm text-muted-foreground uppercase">
            Daty
          </h3>
          <div className="grid grid-cols-2 gap-2 text-sm">
            <span className="text-muted-foreground">Utworzono:</span>
            <span>{formatDate(doc.created_at)}</span>
            {doc.confirmed_at && (
              <>
                <span className="text-muted-foreground">Zatwierdzono:</span>
                <span>{formatDate(doc.confirmed_at)}</span>
              </>
            )}
            {doc.notes && (
              <>
                <span className="text-muted-foreground">Uwagi:</span>
                <span>{doc.notes}</span>
              </>
            )}
          </div>
        </div>
      </div>

      <div className="space-y-4">
        <h3 className="text-lg font-semibold">
          Pozycje ({items.length})
        </h3>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">Brak pozycji</p>
        ) : (
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID Produktu</TableHead>
                  <TableHead>ID Wariantu</TableHead>
                  <TableHead>Ilosc</TableHead>
                  <TableHead>Cena jedn.</TableHead>
                  <TableHead>Uwagi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="font-mono text-xs">
                      {item.product_id}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {item.variant_id || "---"}
                    </TableCell>
                    <TableCell>{item.quantity}</TableCell>
                    <TableCell>
                      {item.unit_price != null
                        ? `${item.unit_price.toFixed(2)} PLN`
                        : "---"}
                    </TableCell>
                    <TableCell>{item.notes || "---"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </AdminGuard>
  );
}
