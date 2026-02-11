"use client";

import { useState } from "react";
import Link from "next/link";
import { Package, Plus, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DataTable } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { EmptyState } from "@/components/shared/empty-state";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useProducts } from "@/hooks/use-products";
import { useProductCategories } from "@/hooks/use-product-categories";
import { formatCurrency, formatDate } from "@/lib/utils";
import { ORDER_SOURCE_LABELS } from "@/lib/constants";
import type { Product } from "@/types/api";

const DEFAULT_LIMIT = 20;

export default function ProductsPage() {
  const [search, setSearch] = useState("");
  const [tagFilter, setTagFilter] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("");
  const [pagination, setPagination] = useState({ limit: DEFAULT_LIMIT, offset: 0 });
  const [sortBy, setSortBy] = useState<string>("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");

  const handleSort = (column: string) => {
    if (sortBy === column) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortBy(column);
      setSortOrder("desc");
    }
    setPagination((prev) => ({ ...prev, offset: 0 }));
  };

  const { data: categoriesConfig } = useProductCategories();

  const { data, isLoading, isError, refetch } = useProducts({
    ...pagination,
    name: search || undefined,
    tag: tagFilter || undefined,
    category: categoryFilter || undefined,
    sort_by: sortBy,
    sort_order: sortOrder,
  });

  const columns = [
    {
      header: "",
      accessorKey: "image_url" as const,
      cell: (product: Product) => (
        product.image_url ? (
          <img
            src={product.image_url}
            alt={product.name}
            className="h-10 w-10 rounded object-cover"
            onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
          />
        ) : (
          <div className="flex h-10 w-10 items-center justify-center rounded bg-muted">
            <Package className="h-5 w-5 text-muted-foreground" />
          </div>
        )
      ),
    },
    {
      header: "Nazwa",
      accessorKey: "name" as const,
      sortable: true,
      cell: (product: Product) => (
        <Link
          href={`/products/${product.id}`}
          className="font-medium text-primary hover:underline"
        >
          {product.name}
        </Link>
      ),
    },
    {
      header: "SKU",
      accessorKey: "sku" as const,
      sortable: true,
      cell: (product: Product) => (
        <span className="font-mono text-sm">{product.sku || "-"}</span>
      ),
    },
    {
      header: "Cena",
      accessorKey: "price" as const,
      sortable: true,
      cell: (product: Product) => (
        <span className="text-sm">{formatCurrency(product.price)}</span>
      ),
    },
    {
      header: "Stan",
      accessorKey: "stock_quantity" as const,
      sortable: true,
      cell: (product: Product) => (
        <span
          className={`text-sm font-medium ${
            product.stock_quantity === 0
              ? "text-destructive"
              : product.stock_quantity < 10
                ? "text-yellow-600 dark:text-yellow-400"
                : ""
          }`}
        >
          {product.stock_quantity}
        </span>
      ),
    },
    {
      header: "Źródło",
      accessorKey: "source" as const,
      cell: (product: Product) => (
        <span className="text-sm">
          {ORDER_SOURCE_LABELS[product.source] ?? product.source}
        </span>
      ),
    },
    {
      header: "Kategoria",
      accessorKey: "category" as const,
      cell: (product: Product) => {
        if (!product.category) return null;
        const cat = categoriesConfig?.categories?.find((c) => c.key === product.category);
        return (
          <span
            className="rounded-full px-2 py-0.5 text-xs font-medium"
            style={{
              backgroundColor: cat?.color ? `${cat.color}20` : undefined,
              color: cat?.color,
            }}
          >
            {cat?.label || product.category}
          </span>
        );
      },
    },
    {
      header: "Tagi",
      accessorKey: "tags" as const,
      cell: (product: Product) => (
        <div className="flex flex-wrap gap-1">
          {product.tags?.map((tag) => (
            <span key={tag} className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
              {tag}
            </span>
          ))}
        </div>
      ),
    },
    {
      header: "Data utworzenia",
      accessorKey: "created_at" as const,
      sortable: true,
      cell: (product: Product) => (
        <span className="text-sm text-muted-foreground">
          {formatDate(product.created_at)}
        </span>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Produkty</h1>
          <p className="text-muted-foreground">
            Zarządzaj katalogiem produktów
          </p>
        </div>
        <Button asChild>
          <Link href="/products/new">
            <Plus className="h-4 w-4" />
            Nowy produkt
          </Link>
        </Button>
      </div>

      <div className="flex items-center gap-4">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Szukaj po nazwie..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPagination((prev) => ({ ...prev, offset: 0 }));
            }}
            className="pl-9"
          />
        </div>
        <Input
          placeholder="Filtruj po tagu..."
          value={tagFilter}
          onChange={(e) => {
            setTagFilter(e.target.value);
            setPagination((prev) => ({ ...prev, offset: 0 }));
          }}
          className="w-[180px]"
        />
        <Select
          value={categoryFilter}
          onValueChange={(value) => {
            setCategoryFilter(value === "__all__" ? "" : value);
            setPagination((prev) => ({ ...prev, offset: 0 }));
          }}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Kategoria..." />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">Wszystkie kategorie</SelectItem>
            {categoriesConfig?.categories
              ?.sort((a, b) => a.position - b.position)
              .map((cat) => (
                <SelectItem key={cat.key} value={cat.key}>
                  {cat.label}
                </SelectItem>
              ))}
          </SelectContent>
        </Select>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystąpił błąd podczas ładowania danych. Spróbuj odświeżyć stronę.
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
      )}

      <DataTable
        columns={columns}
        data={data?.items ?? []}
        isLoading={isLoading}
        emptyState={
          <EmptyState
            icon={Package}
            title="Brak produktów"
            description="Nie znaleziono produktów do wyświetlenia."
            action={{ label: "Nowy produkt", href: "/products/new" }}
          />
        }
        sortBy={sortBy}
        sortOrder={sortOrder}
        onSort={handleSort}
      />

      {data && (
        <DataTablePagination
          total={data.total}
          limit={data.limit}
          offset={data.offset}
          onPageChange={(offset) =>
            setPagination((prev) => ({ ...prev, offset }))
          }
          onPageSizeChange={(limit) =>
            setPagination({ limit, offset: 0 })
          }
        />
      )}
    </div>
  );
}
