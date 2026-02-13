"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  Play,
  Pause,
  Package,
  Search,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAllegroOffers,
  useDeactivateAllegroOffer,
  useActivateAllegroOffer,
  useUpdateAllegroOfferStock,
  useUpdateAllegroOfferPrice,
} from "@/hooks/use-allegro";
import type { AllegroOffer } from "@/hooks/use-allegro";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const PAGE_SIZE = 20;

function statusBadgeVariant(status?: string) {
  switch (status) {
    case "ACTIVE":
      return "default" as const;
    case "INACTIVE":
      return "secondary" as const;
    case "ENDED":
      return "destructive" as const;
    default:
      return "outline" as const;
  }
}

function statusLabel(status?: string) {
  switch (status) {
    case "ACTIVE":
      return "Aktywna";
    case "INACTIVE":
      return "Nieaktywna";
    case "ENDED":
      return "Zakonczona";
    default:
      return status ?? "---";
  }
}

export default function AllegroOffersPage() {
  const [page, setPage] = useState(0);
  const [searchName, setSearchName] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");

  const queryParams = {
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
    name: searchName || undefined,
    publication_status: statusFilter !== "all" ? statusFilter : undefined,
  };

  const { data, isLoading, isFetching } = useAllegroOffers(queryParams);

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/integrations/allegro">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Oferty Allegro</h1>
            <p className="text-muted-foreground">
              Zarzadzaj swoimi ofertami na Allegro
            </p>
          </div>
        </div>

        {/* Filters */}
        <Card>
          <CardContent className="pt-6">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
              <div className="flex-1">
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    placeholder="Szukaj po nazwie..."
                    value={searchName}
                    onChange={(e) => {
                      setSearchName(e.target.value);
                      setPage(0);
                    }}
                    className="pl-10"
                  />
                </div>
              </div>
              <Select
                value={statusFilter}
                onValueChange={(v) => {
                  setStatusFilter(v);
                  setPage(0);
                }}
              >
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="Status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Wszystkie</SelectItem>
                  <SelectItem value="ACTIVE">Aktywne</SelectItem>
                  <SelectItem value="INACTIVE">Nieaktywne</SelectItem>
                  <SelectItem value="ENDED">Zakonczone</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* Offers table */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Package className="h-5 w-5" />
              Oferty
              {data && (
                <span className="text-sm font-normal text-muted-foreground">
                  ({data.totalCount} lacznie)
                </span>
              )}
              {isFetching && (
                <Loader2 className="ml-2 h-4 w-4 animate-spin" />
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Skeleton key={i} className="h-16 w-full" />
                ))}
              </div>
            ) : !data?.offers?.length ? (
              <p className="py-8 text-center text-muted-foreground">
                Brak ofert do wyswietlenia
              </p>
            ) : (
              <>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-16">Zdjecie</TableHead>
                      <TableHead>Nazwa</TableHead>
                      <TableHead className="w-28">Cena</TableHead>
                      <TableHead className="w-24">Stan</TableHead>
                      <TableHead className="w-28">Status</TableHead>
                      <TableHead className="w-40 text-right">Akcje</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.offers.map((offer) => (
                      <OfferRow key={offer.id} offer={offer} />
                    ))}
                  </TableBody>
                </Table>

                {/* Pagination */}
                <div className="mt-4 flex items-center justify-between">
                  <p className="text-sm text-muted-foreground">
                    Strona {page + 1} z{" "}
                    {Math.max(1, Math.ceil(data.totalCount / PAGE_SIZE))}
                  </p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={page === 0}
                      onClick={() => setPage((p) => p - 1)}
                    >
                      Poprzednia
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={(page + 1) * PAGE_SIZE >= data.totalCount}
                      onClick={() => setPage((p) => p + 1)}
                    >
                      Nastepna
                    </Button>
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}

function OfferRow({ offer }: { offer: AllegroOffer }) {
  const [editingStock, setEditingStock] = useState(false);
  const [editingPrice, setEditingPrice] = useState(false);
  const [stockValue, setStockValue] = useState(
    String(offer.stock?.available ?? 0)
  );
  const [priceValue, setPriceValue] = useState(
    offer.sellingMode?.price?.amount ?? "0"
  );

  const deactivate = useDeactivateAllegroOffer();
  const activate = useActivateAllegroOffer();
  const updateStock = useUpdateAllegroOfferStock();
  const updatePrice = useUpdateAllegroOfferPrice();

  const isActive = offer.publication?.status === "ACTIVE";

  const handleToggleStatus = () => {
    if (isActive) {
      deactivate.mutate(offer.id, {
        onSuccess: () => toast.success("Oferta dezaktywowana"),
        onError: () => toast.error("Nie udalo sie dezaktywowac oferty"),
      });
    } else {
      activate.mutate(offer.id, {
        onSuccess: () => toast.success("Oferta aktywowana"),
        onError: () => toast.error("Nie udalo sie aktywowac oferty"),
      });
    }
  };

  const handleSaveStock = () => {
    const qty = parseInt(stockValue, 10);
    if (isNaN(qty) || qty < 0) {
      toast.error("Nieprawidlowa wartosc stanu");
      return;
    }
    updateStock.mutate(
      { offerId: offer.id, quantity: qty },
      {
        onSuccess: () => {
          toast.success("Stan zaktualizowany");
          setEditingStock(false);
        },
        onError: () => toast.error("Nie udalo sie zaktualizowac stanu"),
      }
    );
  };

  const handleSavePrice = () => {
    const amt = parseFloat(priceValue);
    if (isNaN(amt) || amt < 0) {
      toast.error("Nieprawidlowa wartosc ceny");
      return;
    }
    updatePrice.mutate(
      {
        offerId: offer.id,
        amount: amt,
        currency: offer.sellingMode?.price?.currency ?? "PLN",
      },
      {
        onSuccess: () => {
          toast.success("Cena zaktualizowana");
          setEditingPrice(false);
        },
        onError: () => toast.error("Nie udalo sie zaktualizowac ceny"),
      }
    );
  };

  return (
    <TableRow>
      {/* Image */}
      <TableCell>
        {offer.primaryImage?.url ? (
          <img
            src={offer.primaryImage.url}
            alt={offer.name}
            className="h-12 w-12 rounded object-cover"
          />
        ) : (
          <div className="flex h-12 w-12 items-center justify-center rounded bg-muted">
            <Package className="h-5 w-5 text-muted-foreground" />
          </div>
        )}
      </TableCell>

      {/* Name */}
      <TableCell>
        <p className="font-medium text-sm line-clamp-2">{offer.name}</p>
        <p className="text-xs text-muted-foreground">ID: {offer.id}</p>
      </TableCell>

      {/* Price */}
      <TableCell>
        {editingPrice ? (
          <div className="flex items-center gap-1">
            <Input
              value={priceValue}
              onChange={(e) => setPriceValue(e.target.value)}
              className="h-8 w-20 text-sm"
              type="number"
              step="0.01"
              min="0"
            />
            <Button
              size="sm"
              variant="ghost"
              className="h-8 px-2"
              onClick={handleSavePrice}
              disabled={updatePrice.isPending}
            >
              {updatePrice.isPending ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                "OK"
              )}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              className="h-8 px-2"
              onClick={() => setEditingPrice(false)}
            >
              X
            </Button>
          </div>
        ) : (
          <button
            className="text-sm hover:underline cursor-pointer text-left"
            onClick={() => setEditingPrice(true)}
          >
            {offer.sellingMode?.price
              ? `${offer.sellingMode.price.amount} ${offer.sellingMode.price.currency}`
              : "---"}
          </button>
        )}
      </TableCell>

      {/* Stock */}
      <TableCell>
        {editingStock ? (
          <div className="flex items-center gap-1">
            <Input
              value={stockValue}
              onChange={(e) => setStockValue(e.target.value)}
              className="h-8 w-16 text-sm"
              type="number"
              min="0"
            />
            <Button
              size="sm"
              variant="ghost"
              className="h-8 px-2"
              onClick={handleSaveStock}
              disabled={updateStock.isPending}
            >
              {updateStock.isPending ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                "OK"
              )}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              className="h-8 px-2"
              onClick={() => setEditingStock(false)}
            >
              X
            </Button>
          </div>
        ) : (
          <button
            className="text-sm hover:underline cursor-pointer"
            onClick={() => setEditingStock(true)}
          >
            {offer.stock?.available ?? "---"}
          </button>
        )}
      </TableCell>

      {/* Status */}
      <TableCell>
        <Badge variant={statusBadgeVariant(offer.publication?.status)}>
          {statusLabel(offer.publication?.status)}
        </Badge>
      </TableCell>

      {/* Actions */}
      <TableCell className="text-right">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleToggleStatus}
          disabled={
            deactivate.isPending ||
            activate.isPending ||
            offer.publication?.status === "ENDED"
          }
        >
          {deactivate.isPending || activate.isPending ? (
            <Loader2 className="mr-1 h-3 w-3 animate-spin" />
          ) : isActive ? (
            <Pause className="mr-1 h-3 w-3" />
          ) : (
            <Play className="mr-1 h-3 w-3" />
          )}
          {isActive ? "Dezaktywuj" : "Aktywuj"}
        </Button>
      </TableCell>
    </TableRow>
  );
}
