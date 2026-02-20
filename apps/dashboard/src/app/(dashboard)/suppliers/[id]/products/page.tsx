"use client";

import { useState, useRef, useEffect, useMemo } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft, Search, Link2, Package, Download } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useSupplier,
  useSupplierProducts,
  useImportSupplierProducts,
} from "@/hooks/use-suppliers";
import { formatCurrency } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { DataTable, type ColumnDef } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { DensityToggle } from "@/components/shared/density-toggle";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import type { SupplierProduct } from "@/types/api";

const DEFAULT_LIMIT = 50;

export default function SupplierProductsPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;

  const { data: supplier, isLoading: supplierLoading } = useSupplier(id);
  const importProducts = useImportSupplierProducts(id);

  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [linkedFilter, setLinkedFilter] = useState<string>("all");
  const [pagination, setPagination] = useState({ limit: DEFAULT_LIMIT, offset: 0 });
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const handleSearchChange = (value: string) => {
    setSearch(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setDebouncedSearch(value);
      setPagination((prev) => ({ ...prev, offset: 0 }));
    }, 300);
  };

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  const linked = linkedFilter === "linked" ? true : linkedFilter === "unlinked" ? false : undefined;

  const { data, isLoading } = useSupplierProducts(id, {
    ...pagination,
    search: debouncedSearch || undefined,
    linked,
  });

  const items = data?.items ?? [];
  const total = data?.total ?? 0;

  const columns: ColumnDef<SupplierProduct>[] = useMemo(
    () => [
      {
        header: "Nazwa",
        accessorKey: "name",
        cell: (row) => (
          <span className="font-medium max-w-[300px] truncate block">{row.name}</span>
        ),
      },
      {
        header: "EAN",
        accessorKey: "ean",
        cell: (row) => <span className="text-muted-foreground">{row.ean || "---"}</span>,
      },
      {
        header: "SKU",
        accessorKey: "sku",
        cell: (row) => <span className="text-muted-foreground">{row.sku || "---"}</span>,
      },
      {
        header: "Kategoria",
        accessorKey: "source_category",
        cell: (row) => (
          <span className="text-muted-foreground text-xs">
            {row.source_category || "---"}
          </span>
        ),
      },
      {
        header: "Cena",
        accessorKey: "price",
        className: "text-right",
        cell: (row) => (
          <span className="text-right block">
            {row.price != null ? formatCurrency(row.price) : "---"}
          </span>
        ),
      },
      {
        header: "Stan",
        accessorKey: "stock_quantity",
        className: "text-right",
        cell: (row) => <span className="text-right block">{row.stock_quantity}</span>,
      },
      {
        header: "Status",
        accessorKey: "product_id",
        cell: (row) =>
          row.product_id ? (
            <Badge variant="outline" className="gap-1">
              <Link2 className="h-3 w-3" />
              Powiązany
            </Badge>
          ) : (
            <Badge variant="secondary">Niepowiązany</Badge>
          ),
      },
    ],
    []
  );

  const handleImport = () => {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) return;

    importProducts.mutate(
      { supplier_product_ids: ids },
      {
        onSuccess: (result) => {
          const parts: string[] = [];
          if (result.imported > 0) parts.push(`Zaimportowano: ${result.imported}`);
          if (result.skipped > 0) parts.push(`Pominięto: ${result.skipped}`);
          if (result.errors?.length) parts.push(`Błędy: ${result.errors.length}`);
          toast.success(parts.join(", ") || "Import zakończony");
          setSelectedIds(new Set());
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  // Count only unlinked in selection for the import button label
  const selectedUnlinked = useMemo(() => {
    if (selectedIds.size === 0) return 0;
    return items.filter((sp) => selectedIds.has(sp.id) && !sp.product_id).length;
  }, [selectedIds, items]);

  if (supplierLoading) return <LoadingSkeleton />;
  if (!supplier) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        Dostawca nie znaleziony
      </div>
    );
  }

  return (
    <AdminGuard>
      <div className="space-y-4">
        {/* Header */}
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => router.push(`/suppliers/${id}`)}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div className="flex-1">
            <h1 className="text-2xl font-bold tracking-tight">
              Produkty dostawcy: {supplier.name}
            </h1>
            <p className="text-muted-foreground text-sm">
              {total} produktów w katalogu
            </p>
          </div>
        </div>

        {/* Toolbar */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="relative flex-1 min-w-[200px] max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => handleSearchChange(e.target.value)}
              placeholder="Szukaj po nazwie, EAN, SKU..."
              className="pl-9"
            />
          </div>
          <Select
            value={linkedFilter}
            onValueChange={(value) => {
              setLinkedFilter(value);
              setPagination((prev) => ({ ...prev, offset: 0 }));
            }}
          >
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Status powiązania" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Wszystkie</SelectItem>
              <SelectItem value="unlinked">Niepowiązane</SelectItem>
              <SelectItem value="linked">Powiązane</SelectItem>
            </SelectContent>
          </Select>
          <DensityToggle />
          {selectedIds.size > 0 && (
            <Button
              onClick={handleImport}
              disabled={importProducts.isPending || selectedUnlinked === 0}
            >
              <Download className="h-4 w-4 mr-2" />
              {importProducts.isPending
                ? "Importowanie..."
                : `Dodaj do produktów (${selectedUnlinked})`}
            </Button>
          )}
        </div>

        {/* Table */}
        <DataTable
          columns={columns}
          data={items}
          isLoading={isLoading}
          selectable
          selectedIds={selectedIds}
          onSelectionChange={setSelectedIds}
          rowId={(row) => row.id}
          emptyMessage="Brak produktów. Uruchom synchronizację feeda."
        />

        {/* Pagination */}
        {total > 0 && (
          <DataTablePagination
            total={total}
            limit={pagination.limit}
            offset={pagination.offset}
            onPageChange={(offset) => setPagination((prev) => ({ ...prev, offset }))}
            onPageSizeChange={(limit) => setPagination({ limit, offset: 0 })}
          />
        )}
      </div>
    </AdminGuard>
  );
}
