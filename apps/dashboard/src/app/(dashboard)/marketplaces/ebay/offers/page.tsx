"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { ArrowLeft, Loader2, Package, Search } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useIntegrations } from "@/hooks/use-integrations";
import { apiClient } from "@/lib/api-client";
import { useQuery } from "@tanstack/react-query";
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
import { EbayTabNav } from "../_components/ebay-tab-nav";
import { useTranslations } from "next-intl";

interface EbayOffer {
  offerId: string;
  sku: string;
  marketplaceId: string;
  format: string;
  listingDescription?: string;
  availableQuantity?: number;
  categoryId: string;
  pricingSummary?: {
    price: { value: string; currency: string };
  };
  status?: string;
  listing?: { listingId?: string };
}

interface EbayOffersResponse {
  offers: EbayOffer[];
  total: number;
  size: number;
  next?: string;
}

const PAGE_SIZE = 20;

function statusBadgeVariant(status?: string) {
  switch (status) {
    case "PUBLISHED":
      return "default" as const;
    case "ACTIVE":
      return "default" as const;
    case "UNPUBLISHED":
      return "secondary" as const;
    case "ENDED":
      return "destructive" as const;
    default:
      return "outline" as const;
  }
}

function statusLabel(status?: string) {
  switch (status) {
    case "PUBLISHED":
      return "Opublikowana";
    case "ACTIVE":
      return "Aktywna";
    case "UNPUBLISHED":
      return "Nieopublikowana";
    case "ENDED":
      return "Zakonczona";
    default:
      return status ?? "---";
  }
}

function useEbayOffers(
  integrationId: string | undefined,
  params?: { limit?: number; offset?: number; sku?: string }
) {
  const searchParams = new URLSearchParams();
  if (integrationId) searchParams.set("integration_id", integrationId);
  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));
  if (params?.sku) searchParams.set("sku", params.sku);
  const qs = searchParams.toString();

  return useQuery({
    queryKey: ["ebay", "offers", integrationId, params],
    queryFn: () =>
      apiClient<EbayOffersResponse>(
        `/v1/integrations/ebay/offers${qs ? `?${qs}` : ""}`
      ),
    enabled: !!integrationId,
  });
}

export default function EbayOffersPage() {
  const t = useTranslations("marketplaces");
  const { data: integrations, isLoading: integrationsLoading } =
    useIntegrations();

  const ebay = useMemo(
    () => integrations?.find((i) => i.provider === "ebay") ?? null,
    [integrations]
  );

  const [page, setPage] = useState(0);
  const [searchSku, setSearchSku] = useState("");

  const queryParams = {
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
    sku: searchSku || undefined,
  };

  const { data, isLoading, isFetching, error } = useEbayOffers(
    ebay?.id,
    queryParams
  );

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/marketplaces/ebay">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">eBay</h1>
            <p className="text-muted-foreground">
              {t("zarzadzajOfertamiIKonfiguracjaEbay")}
            </p>
          </div>
        </div>

        <EbayTabNav />

        {integrationsLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        ) : !ebay ? (
          <Card>
            <CardContent className="pt-6">
              <p className="py-8 text-center text-muted-foreground">
                Brak skonfigurowanej integracji eBay.{" "}
                <Link
                  href="/marketplaces/ebay"
                  className="text-primary underline"
                >
                  {t("skonfigurujIntegracje")}
                </Link>
              </p>
            </CardContent>
          </Card>
        ) : (
          <>
            {error && (
              <Card className="border-destructive/50">
                <CardContent className="pt-6">
                  <p className="text-sm text-destructive">
                    {error instanceof Error
                      ? error.message
                      : t("nieudałosiepobracofertzebay")}
                  </p>
                </CardContent>
              </Card>
            )}

            {/* Filters */}
            <Card>
              <CardContent className="pt-6">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
                  <div className="flex-1">
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        placeholder="Szukaj po SKU..."
                        value={searchSku}
                        onChange={(e) => {
                          setSearchSku(e.target.value);
                          setPage(0);
                        }}
                        className="pl-10"
                      />
                    </div>
                  </div>
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
                      ({data.total} łącznie)
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
                    {t("brakOfertDoWyswietlenia")}
                  </p>
                ) : (
                  <>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>SKU</TableHead>
                          <TableHead>Offer ID</TableHead>
                          <TableHead>Listing ID</TableHead>
                          <TableHead className="w-28">Marketplace</TableHead>
                          <TableHead className="w-28">Cena</TableHead>
                          <TableHead className="w-24">Stan</TableHead>
                          <TableHead className="w-28">Status</TableHead>
                          <TableHead className="w-24">Format</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {data.offers.map((offer) => (
                          <OfferRow key={offer.offerId} offer={offer} />
                        ))}
                      </TableBody>
                    </Table>

                    {/* Pagination */}
                    <div className="mt-4 flex items-center justify-between">
                      <p className="text-sm text-muted-foreground">
                        Strona {page + 1} z{" "}
                        {Math.max(1, Math.ceil(data.total / PAGE_SIZE))}
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
                          disabled={(page + 1) * PAGE_SIZE >= data.total}
                          onClick={() => setPage((p) => p + 1)}
                        >
                          {t("next")}
                        </Button>
                      </div>
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </AdminGuard>
  );
}

function OfferRow({ offer }: { offer: EbayOffer }) {
  return (
    <TableRow>
      {/* SKU */}
      <TableCell>
        <p className="font-medium text-sm">{offer.sku}</p>
      </TableCell>

      {/* Offer ID */}
      <TableCell>
        <p className="font-mono text-xs text-muted-foreground">
          {offer.offerId || "---"}
        </p>
      </TableCell>

      {/* Listing ID */}
      <TableCell>
        <p className="font-mono text-xs text-muted-foreground">
          {offer.listing?.listingId || "---"}
        </p>
      </TableCell>

      {/* Marketplace */}
      <TableCell>
        <p className="text-sm">{offer.marketplaceId || "---"}</p>
      </TableCell>

      {/* Price */}
      <TableCell>
        <p className="text-sm">
          {offer.pricingSummary?.price
            ? `${offer.pricingSummary.price.value} ${offer.pricingSummary.price.currency}`
            : "---"}
        </p>
      </TableCell>

      {/* Stock */}
      <TableCell>
        <p className="text-sm">{offer.availableQuantity ?? "---"}</p>
      </TableCell>

      {/* Status */}
      <TableCell>
        <Badge variant={statusBadgeVariant(offer.status)}>
          {statusLabel(offer.status)}
        </Badge>
      </TableCell>

      {/* Format */}
      <TableCell>
        <p className="text-sm text-muted-foreground">
          {offer.format || "---"}
        </p>
      </TableCell>
    </TableRow>
  );
}
