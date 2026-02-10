"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, ShoppingBag, Store } from "lucide-react";
import { useProduct } from "@/hooks/use-products";
import { useProductListings } from "@/hooks/use-product-listings";
import { EmptyState } from "@/components/shared/empty-state";
import { StatusBadge } from "@/components/shared/status-badge";
import { formatDate } from "@/lib/utils";
import { INTEGRATION_PROVIDER_LABELS } from "@/lib/constants";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const LISTING_STATUSES: Record<string, { label: string; color: string }> = {
  active: { label: "Aktywna", color: "bg-green-100 text-green-800" },
  inactive: { label: "Nieaktywna", color: "bg-gray-100 text-gray-800" },
  draft: { label: "Szkic", color: "bg-yellow-100 text-yellow-800" },
  ended: { label: "Zakończona", color: "bg-red-100 text-red-800" },
};

const SYNC_STATUSES: Record<string, { label: string; color: string }> = {
  synced: { label: "Zsynchronizowana", color: "bg-green-100 text-green-800" },
  pending: { label: "Oczekuje", color: "bg-yellow-100 text-yellow-800" },
  error: { label: "Błąd", color: "bg-red-100 text-red-800" },
  never: { label: "Nigdy", color: "bg-gray-100 text-gray-800" },
};

export default function ProductListingsPage() {
  const params = useParams<{ id: string }>();
  const { data: product, isLoading: productLoading } = useProduct(params.id);
  const { data: listings, isLoading: listingsLoading } = useProductListings(
    params.id
  );

  if (productLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!product) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Nie znaleziono produktu</h1>
        <Button asChild variant="outline">
          <Link href="/products">Wróć do listy</Link>
        </Button>
      </div>
    );
  }

  const hasListings = listings && listings.length > 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href={`/products/${params.id}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">{product.name}</h1>
            <p className="text-muted-foreground">Oferty marketplace</p>
          </div>
        </div>
        <Button asChild>
          <Link href="/integrations/allegro">
            <Store className="mr-2 h-4 w-4" />
            Dodaj do marketplace
          </Link>
        </Button>
      </div>

      {listingsLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : !hasListings ? (
        <EmptyState
          icon={ShoppingBag}
          title="Brak ofert marketplace"
          description="Ten produkt nie jest jeszcze wystawiony na żadnym marketplace. Połącz integrację, aby rozpocząć sprzedaż."
          action={{
            label: "Połącz Allegro",
            href: "/integrations/allegro",
          }}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Marketplace</TableHead>
                <TableHead>ID zewnętrzne</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Synchronizacja</TableHead>
                <TableHead>Ostatnia synchronizacja</TableHead>
                <TableHead className="w-[80px]">Akcje</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {listings.map((listing) => (
                <TableRow key={listing.id}>
                  <TableCell className="font-medium">
                    {INTEGRATION_PROVIDER_LABELS[listing.metadata?.provider as string] ??
                      "Marketplace"}
                  </TableCell>
                  <TableCell className="font-mono text-sm">
                    {listing.external_id || "---"}
                  </TableCell>
                  <TableCell>
                    <StatusBadge
                      status={listing.status}
                      statusMap={LISTING_STATUSES}
                    />
                  </TableCell>
                  <TableCell>
                    <StatusBadge
                      status={listing.sync_status}
                      statusMap={SYNC_STATUSES}
                    />
                  </TableCell>
                  <TableCell>
                    {listing.last_synced_at
                      ? formatDate(listing.last_synced_at)
                      : "---"}
                  </TableCell>
                  <TableCell>
                    {listing.url && (
                      <Button variant="ghost" size="sm" asChild>
                        <a
                          href={listing.url}
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          Zobacz
                        </a>
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {hasListings &&
        listings.some((l) => l.error_message) && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium text-destructive">
              Błędy synchronizacji
            </h3>
            {listings
              .filter((l) => l.error_message)
              .map((l) => (
                <div
                  key={l.id}
                  className="rounded-md border border-destructive/50 bg-destructive/10 p-3"
                >
                  <p className="text-sm text-destructive">{l.error_message}</p>
                </div>
              ))}
          </div>
        )}
    </div>
  );
}
