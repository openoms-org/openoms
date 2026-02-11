"use client";

import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useWarehouseDocument,
  useConfirmWarehouseDocument,
  useCancelWarehouseDocument,
} from "@/hooks/use-warehouse-documents";
import { useWarehouse } from "@/hooks/use-warehouses";
import { useSupplier } from "@/hooks/use-suppliers";
import { useOrder } from "@/hooks/use-orders";
import { useProduct } from "@/hooks/use-products";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate, shortId } from "@/lib/utils";
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

function ProductName({ productId }: { productId: string }) {
  const { data: product } = useProduct(productId);
  if (product) return <>{product.name}</>;
  return <span className="font-mono text-xs">{shortId(productId)}</span>;
}

const DOC_TYPE_LABELS: Record<string, string> = {
  PZ: "PZ - Przyjęcie zewnętrzne",
  WZ: "WZ - Wydanie zewnętrzne",
  MM: "MM - Przesunięcie międzymagazynowe",
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

  const { data: warehouse } = useWarehouse(doc?.warehouse_id ?? "");
  const { data: targetWarehouse } = useWarehouse(doc?.target_warehouse_id ?? "");
  const { data: supplier } = useSupplier(doc?.supplier_id ?? "");
  const { data: order } = useOrder(doc?.order_id ?? "");

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError || !doc) {
    return (
      <AdminGuard>
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Nie udało się załadować dokumentu.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Spróbuj ponownie
          </Button>
        </div>
      </AdminGuard>
    );
  }

  const handleConfirm = () => {
    confirmDoc.mutate(id, {
      onSuccess: () => {
        toast.success("Dokument został zatwierdzony. Stany magazynowe zaktualizowane.");
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
        toast.success("Dokument został anulowany");
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
            Powrót do listy
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
                  {confirmDoc.isPending ? "Zatwierdzanie..." : "Zatwierdź"}
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
            Szczegóły
          </h3>
          <div className="grid grid-cols-2 gap-2 text-sm">
            <span className="text-muted-foreground">Typ:</span>
            <span>{doc.document_type}</span>
            <span className="text-muted-foreground">Magazyn:</span>
            <span>{warehouse?.name ?? shortId(doc.warehouse_id)}</span>
            {doc.target_warehouse_id && (
              <>
                <span className="text-muted-foreground">Magazyn docelowy:</span>
                <span>
                  {targetWarehouse?.name ?? shortId(doc.target_warehouse_id)}
                </span>
              </>
            )}
            {doc.supplier_id && (
              <>
                <span className="text-muted-foreground">Dostawca:</span>
                <span>{supplier?.name ?? shortId(doc.supplier_id)}</span>
              </>
            )}
            {doc.order_id && (
              <>
                <span className="text-muted-foreground">Zamówienie:</span>
                <Link href={`/orders/${doc.order_id}`} className="text-primary hover:underline">
                  {order ? `#${shortId(doc.order_id)}` : shortId(doc.order_id)}
                </Link>
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
                  <TableHead>Produkt</TableHead>
                  <TableHead>Wariant</TableHead>
                  <TableHead>Ilość</TableHead>
                  <TableHead>Cena jedn.</TableHead>
                  <TableHead>Uwagi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <ProductName productId={item.product_id} />
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
