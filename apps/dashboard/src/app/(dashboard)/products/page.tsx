"use client";

import { useState } from "react";
import Link from "next/link";
import { Package, Plus, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DataTable } from "@/components/shared/data-table";
import { DataTablePagination } from "@/components/shared/data-table-pagination";
import { useProducts } from "@/hooks/use-products";
import { formatCurrency, formatDate } from "@/lib/utils";
import type { Product } from "@/types/api";

const DEFAULT_LIMIT = 20;

const SOURCE_LABELS: Record<string, string> = {
  manual: "Reczne",
  allegro: "Allegro",
  woocommerce: "WooCommerce",
};

export default function ProductsPage() {
  const [search, setSearch] = useState("");
  const [pagination, setPagination] = useState({ limit: DEFAULT_LIMIT, offset: 0 });

  const { data, isLoading } = useProducts({
    ...pagination,
    name: search || undefined,
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
      cell: (product: Product) => (
        <span className="font-mono text-sm">{product.sku || "-"}</span>
      ),
    },
    {
      header: "Cena",
      accessorKey: "price" as const,
      cell: (product: Product) => (
        <span className="text-sm">{formatCurrency(product.price)}</span>
      ),
    },
    {
      header: "Stan",
      accessorKey: "stock_quantity" as const,
      cell: (product: Product) => (
        <span
          className={`text-sm font-medium ${
            product.stock_quantity === 0
              ? "text-destructive"
              : product.stock_quantity < 10
                ? "text-yellow-600"
                : ""
          }`}
        >
          {product.stock_quantity}
        </span>
      ),
    },
    {
      header: "Zrodlo",
      accessorKey: "source" as const,
      cell: (product: Product) => (
        <span className="text-sm">
          {SOURCE_LABELS[product.source] ?? product.source}
        </span>
      ),
    },
    {
      header: "Data utworzenia",
      accessorKey: "created_at" as const,
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
            Zarzadzaj katalogiem produktow
          </p>
        </div>
        <Button asChild>
          <Link href="/products/new">
            <Plus className="h-4 w-4" />
            Nowy produkt
          </Link>
        </Button>
      </div>

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

      <DataTable
        columns={columns}
        data={data?.items ?? []}
        isLoading={isLoading}
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
