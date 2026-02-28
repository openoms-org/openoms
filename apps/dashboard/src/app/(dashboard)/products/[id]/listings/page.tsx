"use client";

import { useState, useMemo, useEffect } from "react";
import { useParams, useSearchParams } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import {
  ArrowLeft,
  ChevronRight,
  ExternalLink,
  FolderOpen,
  Info,
  Loader2,
  Package,
  Plus,
  RotateCcw,
  Search,
  Tag,
  Trash2,
  Upload,
} from "lucide-react";
import { DescriptionEditor, plainTextToHTML } from "@/components/editor/description-editor";
import { apiClient } from "@/lib/api-client";
import type { AISuggestion, AITextResult } from "@/types/api";
import { AdminGuard } from "@/components/shared/admin-guard";
import { EmptyState } from "@/components/shared/empty-state";
import { useProduct } from "@/hooks/use-products";
import { useProductSupplierLink, useAllegroParameterMappings } from "@/hooks/use-suppliers";
import { useIntegrations } from "@/hooks/use-integrations";
import {
  useProductListings,
  useCreateProductListing,
  useCreateWooCommerceListing,
  useDeleteProductListing,
  useSyncProductListing,
  useUpdateListingSyncMode,
  useForcePushListing,
  useAllegroCategories,
  useAllegroCategorySearch,
  useAllegroCategoryParams,
  useAllegroShippingRates,
  useAllegroReturnPolicies,
  useAllegroWarranties,
  useCreateAllegroReturnPolicy,
  useCreateAllegroWarranty,
  useAutoGenerateShippingRate,
  useAllegroProductSearch,
  useAllegroListingSearch,
} from "@/hooks/use-allegro";
import type {
  AllegroCategory,
  AllegroCategoryParameter,
  AllegroMatchingCategory,
  CreateProductListingRequest,
} from "@/hooks/use-allegro";
import Image from "next/image";
import type { Product, ProductListing, Integration } from "@/types/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { AlertTriangle, Check, ChevronsUpDown } from "lucide-react";
import { sanitizeUrl } from "@/lib/utils";

// ===================== Constants =====================

const PROVIDER_LABELS: Record<string, string> = {
  allegro: "Allegro",
  woocommerce: "WooCommerce",
  amazon: "Amazon",
  ebay: "eBay",
};

function providerLabel(provider: string): string {
  return PROVIDER_LABELS[provider] ?? provider;
}

const PROVINCES = [
  "DOLNOSLASKIE",
  "KUJAWSKO_POMORSKIE",
  "LUBELSKIE",
  "LUBUSKIE",
  "LODZKIE",
  "MALOPOLSKIE",
  "MAZOWIECKIE",
  "OPOLSKIE",
  "PODKARPACKIE",
  "PODLASKIE",
  "POMORSKIE",
  "SLASKIE",
  "SWIETOKRZYSKIE",
  "WARMINSKO_MAZURSKIE",
  "WIELKOPOLSKIE",
  "ZACHODNIOPOMORSKIE",
] as const;

const PROVINCE_LABELS: Record<string, string> = {
  DOLNOSLASKIE: "Dolnoslaskie",
  KUJAWSKO_POMORSKIE: "Kujawsko-pomorskie",
  LUBELSKIE: "Lubelskie",
  LUBUSKIE: "Lubuskie",
  LODZKIE: "Lodzkie",
  MALOPOLSKIE: "Malopolskie",
  MAZOWIECKIE: "Mazowieckie",
  OPOLSKIE: "Opolskie",
  PODKARPACKIE: "Podkarpackie",
  PODLASKIE: "Podlaskie",
  POMORSKIE: "Pomorskie",
  SLASKIE: "Slaskie",
  SWIETOKRZYSKIE: "Swietokrzyskie",
  WARMINSKO_MAZURSKIE: "Warminsko-mazurskie",
  WIELKOPOLSKIE: "Wielkopolskie",
  ZACHODNIOPOMORSKIE: "Zachodniopomorskie",
};

const HANDLING_TIME_OPTIONS = [
  { value: "PT24H", label: "24 godziny" },
  { value: "PT48H", label: "48 godzin" },
  { value: "PT72H", label: "72 godziny" },
  { value: "PT96H", label: "4 dni" },
  { value: "PT120H", label: "5 dni" },
];

// ===================== Main Page =====================

export default function ProductListingsPage() {
  const params = useParams<{ id: string }>();
  const { data: product } = useProduct(params.id);
  const { data: listings, isLoading } = useProductListings(params.id);

  // Auto-open wizard when redirected with ?listing=new
  const searchParams = useSearchParams();
  const [showCreate, setShowCreate] = useState(false);
  useEffect(() => {
    if (searchParams.get("listing") === "new") {
      setShowCreate(true);
    }
  }, [searchParams]);
  const deleteListing = useDeleteProductListing(params.id);
  const syncListing = useSyncProductListing(params.id);
  const updateSyncMode = useUpdateListingSyncMode(params.id);
  const forcePush = useForcePushListing();
  const [deleteTarget, setDeleteTarget] = useState<ProductListing | null>(null);
  const [showDeleteAll, setShowDeleteAll] = useState(false);
  const [deletingAll, setDeletingAll] = useState(false);
  const { data: integrations } = useIntegrations();

  const providerByIntegration = useMemo(() => {
    const map: Record<string, string> = {};
    for (const intg of integrations ?? []) {
      map[intg.id] = intg.provider;
    }
    return map;
  }, [integrations]);

  return (
    <AdminGuard>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href={`/products/${params.id}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div className="flex-1">
            <h1 className="text-2xl font-bold">Oferty marketplace</h1>
            <p className="text-muted-foreground">
              {product?.name ?? "Ladowanie..."}
            </p>
          </div>
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Wystaw na marketplace
          </Button>
        </div>

        {/* Info section */}
        <div className="rounded-lg border bg-muted/50 p-4 flex gap-3">
          <Info className="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div className="space-y-1 text-sm">
            <p className="font-medium">Jak wystawic produkt na Allegro?</p>
            <ul className="list-disc list-inside space-y-0.5 text-muted-foreground">
              <li>
                Kliknij &quot;Wystaw na Allegro&quot; i przejdz przez
                4-krokowy formularz.
              </li>
              <li>
                Wybierz kategorie Allegro (musisz dotrzec do kategorii koncowej
                oznaczonej &quot;Lisc&quot;).
              </li>
              <li>
                Wypelnij wymagane parametry kategorii (np. rozmiar, kolor,
                marka).
              </li>
              <li>
                Wybierz cennik wysylki, polityke zwrotow i rekojmie — jesli ich
                nie masz, utworz je w sekcji Dostawa i Polityki.
              </li>
              <li>
                Po wystawieniu oferta pojawi sie na Allegro, a stan magazynowy
                bedzie automatycznie synchronizowany co 5 minut.
              </li>
            </ul>
          </div>
        </div>

        {/* Listings table */}
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        ) : !listings?.length ? (
          <EmptyState
            icon={Package}
            title="Brak ofert marketplace"
            description="Ten produkt nie jest jeszcze wystawiony na zadnym marketplace."
          />
        ) : (
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>Aktywne oferty ({listings.length})</CardTitle>
              {listings.length > 1 && (
                <Button
                  variant="outline"
                  size="sm"
                  className="text-destructive"
                  onClick={() => setShowDeleteAll(true)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Usun ze wszystkich
                </Button>
              )}
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Platforma</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>ID oferty</TableHead>
                    <TableHead>Tryb sync</TableHead>
                    <TableHead>Synchronizacja</TableHead>
                    <TableHead>Ostatnia synch.</TableHead>
                    <TableHead className="text-right">Akcje</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {listings.map((listing) => (
                    <ListingRow
                      key={listing.id}
                      listing={listing}
                      providerName={providerLabel(providerByIntegration[listing.integration_id] ?? "allegro")}
                      onSync={(id) =>
                        syncListing.mutate(id, {
                          onSuccess: () =>
                            toast.success("Zsynchronizowano"),
                          onError: () =>
                            toast.error("Blad synchronizacji"),
                        })
                      }
                      onToggleSyncMode={(id, mode) =>
                        updateSyncMode.mutate(
                          { listingId: id, mode },
                          {
                            onSuccess: () =>
                              toast.success(mode === "auto" ? "Tryb automatyczny" : "Tryb reczny"),
                            onError: () =>
                              toast.error("Blad zmiany trybu"),
                          }
                        )
                      }
                      onForcePush={(id) =>
                        forcePush.mutate(id, {
                          onSuccess: () =>
                            toast.success("Stan wyslany do marketplace"),
                          onError: () =>
                            toast.error("Blad wysylania stanu"),
                        })
                      }
                      onDelete={() => setDeleteTarget(listing)}
                    />
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        )}

        {/* Create dialog */}
        {showCreate && product && (
          <CreateListingWizard
            product={product}
            onClose={() => setShowCreate(false)}
          />
        )}

        {/* Delete single listing dialog */}
        <AlertDialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                Usunac oferte z {deleteTarget ? providerLabel(providerByIntegration[deleteTarget.integration_id] ?? "marketplace") : "marketplace"}?
              </AlertDialogTitle>
              <AlertDialogDescription asChild>
                <div className="space-y-3">
                  {deleteTarget?.external_id && (
                    <p>
                      ID oferty:{" "}
                      <span className="font-mono font-medium text-foreground">
                        {deleteTarget.external_id}
                      </span>
                    </p>
                  )}
                  <ul className="list-disc list-inside space-y-1 text-sm">
                    <li>
                      <span className="font-medium text-foreground">
                        Dezaktywowana na {providerLabel(providerByIntegration[deleteTarget?.integration_id ?? ""] ?? "marketplace")}
                      </span>{" "}
                      — oferta nie bedzie juz widoczna dla kupujacych
                    </li>
                    <li>
                      <span className="font-medium text-foreground">Usunieta z OMS</span>{" "}
                      — powiazanie z produktem zostanie trwale skasowane
                    </li>
                  </ul>
                  <p className="text-xs text-muted-foreground">
                    Mozesz pozniej ponownie wystawic ten produkt na ta platforme.
                  </p>
                </div>
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Anuluj</AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                onClick={() => {
                  if (!deleteTarget) return;
                  deleteListing.mutate(deleteTarget.id, {
                    onSuccess: () => {
                      toast.success("Oferta dezaktywowana i usunieta");
                      setDeleteTarget(null);
                    },
                    onError: () =>
                      toast.error("Blad usuwania oferty"),
                  });
                }}
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Usun i dezaktywuj
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {/* Delete all listings dialog */}
        <AlertDialog open={showDeleteAll} onOpenChange={setShowDeleteAll}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-destructive" />
                Usunac oferty ze wszystkich platform?
              </AlertDialogTitle>
              <AlertDialogDescription asChild>
                <div className="space-y-3">
                  <p>
                    Produkt zostanie usuniety z{" "}
                    <span className="font-medium text-foreground">
                      {listings?.length ?? 0} {(listings?.length ?? 0) === 1 ? "platformy" : "platform"}
                    </span>:
                  </p>
                  <ul className="list-disc list-inside space-y-1 text-sm">
                    {listings?.map((l) => (
                      <li key={l.id}>
                        <span className="font-medium text-foreground">
                          {providerLabel(providerByIntegration[l.integration_id] ?? "marketplace")}
                        </span>
                        {l.external_id && (
                          <span className="text-muted-foreground"> (ID: {l.external_id})</span>
                        )}
                        {" "}— oferta zostanie dezaktywowana
                      </li>
                    ))}
                  </ul>
                  <p className="text-xs text-muted-foreground">
                    Wszystkie oferty zostana dezaktywowane na platformach i usuniete z OMS.
                    Mozesz je ponownie wystawic pozniej.
                  </p>
                </div>
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={deletingAll}>Anuluj</AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                disabled={deletingAll}
                onClick={async (e) => {
                  e.preventDefault();
                  if (!listings?.length) return;
                  setDeletingAll(true);
                  let failed = 0;
                  for (const l of listings) {
                    try {
                      await new Promise<void>((resolve, reject) =>
                        deleteListing.mutate(l.id, { onSuccess: () => resolve(), onError: () => reject() })
                      );
                    } catch {
                      failed++;
                    }
                  }
                  setDeletingAll(false);
                  setShowDeleteAll(false);
                  if (failed === 0) {
                    toast.success("Wszystkie oferty usuniete");
                  } else {
                    toast.error(`Nie udalo sie usunac ${failed} z ${listings.length} ofert`);
                  }
                }}
              >
                {deletingAll ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Trash2 className="mr-2 h-4 w-4" />
                )}
                {deletingAll ? "Usuwanie..." : "Usun wszystkie"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </AdminGuard>
  );
}

// ===================== Listing Row =====================

function ListingRow({
  listing,
  providerName,
  onSync,
  onToggleSyncMode,
  onForcePush,
  onDelete,
}: {
  listing: ProductListing;
  providerName: string;
  onSync: (id: string) => void;
  onToggleSyncMode: (id: string, mode: 'auto' | 'manual') => void;
  onForcePush: (id: string) => void;
  onDelete: () => void;
}) {
  const isAuto = listing.stock_sync_mode === "auto";

  return (
    <TableRow>
      <TableCell>
        <Badge variant="outline">{providerName}</Badge>
      </TableCell>
      <TableCell>
        <Badge
          variant={
            listing.status === "active"
              ? "default"
              : listing.status === "inactive"
                ? "secondary"
                : "outline"
          }
        >
          {listing.status === "active"
            ? "Aktywna"
            : listing.status === "inactive"
              ? "Nieaktywna"
              : "Oczekuje"}
        </Badge>
      </TableCell>
      <TableCell className="font-mono text-xs">
        {listing.external_id ?? "---"}
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          <Button
            variant={isAuto ? "default" : "outline"}
            size="sm"
            className="text-xs h-7 px-2"
            onClick={() => onToggleSyncMode(listing.id, isAuto ? "manual" : "auto")}
          >
            {isAuto ? "Auto" : "Reczny"}
          </Button>
          {!isAuto && listing.external_id && (
            <Button
              variant="outline"
              size="sm"
              className="text-xs h-7 px-2"
              onClick={() => onForcePush(listing.id)}
              title="Wymus synchronizacje"
            >
              <Upload className="h-3 w-3 mr-1" />
              Push
            </Button>
          )}
        </div>
      </TableCell>
      <TableCell>
        <Badge
          variant={
            listing.sync_status === "synced"
              ? "default"
              : listing.sync_status === "error"
                ? "destructive"
                : "secondary"
          }
        >
          {listing.sync_status === "synced"
            ? "OK"
            : listing.sync_status === "error"
              ? "Blad"
              : "Oczekuje"}
        </Badge>
        {listing.error_message && (
          <p className="text-xs text-destructive mt-1">
            {listing.error_message}
          </p>
        )}
      </TableCell>
      <TableCell className="text-sm text-muted-foreground">
        {listing.last_synced_at
          ? new Date(listing.last_synced_at).toLocaleString("pl-PL")
          : "---"}
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onSync(listing.id)}
          >
            <RotateCcw className="h-4 w-4" />
          </Button>
          {(listing.url || listing.external_id) && (
            <Button variant="ghost" size="sm" asChild>
              <a
                href={sanitizeUrl(listing.url || `https://allegro.pl/moje-allegro/sprzedaz/oferty/${listing.external_id}`)}
                target="_blank"
                rel="noopener noreferrer"
              >
                <ExternalLink className="h-4 w-4" />
              </a>
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onDelete()}
          >
            <Trash2 className="h-4 w-4 text-destructive" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

// ===================== Marketplace Picker =====================

const MARKETPLACE_PROVIDERS = [
  { key: "allegro", name: "Allegro", logo: "/logos/allegro.svg", description: "Wystawianie ofert na Allegro.pl" },
  { key: "woocommerce", name: "WooCommerce", logo: "/logos/woocommerce.svg", description: "Publikacja produktu w sklepie WooCommerce" },
] as const;

function CreateListingWizard({
  product,
  onClose,
}: {
  product: Product;
  onClose: () => void;
}) {
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const { data: integrations } = useIntegrations();

  // Filter to only providers that have active integrations
  const availableProviders = useMemo(() => {
    if (!integrations) return [];
    const providerSet = new Set(integrations.map((i) => i.provider));
    return MARKETPLACE_PROVIDERS.filter((mp) => providerSet.has(mp.key));
  }, [integrations]);

  // Show picker always — user should explicitly choose the marketplace

  if (selectedProvider === "allegro") {
    return <CreateAllegroListingDialog product={product} onClose={onClose} />;
  }

  if (selectedProvider === "woocommerce") {
    return <CreateWooCommerceListingDialog product={product} onClose={onClose} />;
  }

  // Marketplace picker step
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Wystaw na marketplace</DialogTitle>
          <DialogDescription>
            Wybierz marketplace, na którym chcesz wystawić produkt &quot;{product.name}&quot;
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-3 py-4">
          {availableProviders.length === 0 && (
            <p className="text-sm text-muted-foreground text-center py-4">
              Brak skonfigurowanych integracji marketplace. Przejdź do{" "}
              <Link href="/integrations" className="text-primary hover:underline">
                Integracji
              </Link>
              , aby dodać marketplace.
            </p>
          )}
          {availableProviders.map((mp) => (
            <button
              key={mp.key}
              type="button"
              className="flex items-start gap-4 rounded-xl border p-4 text-left transition-colors hover:bg-muted/50"
              onClick={() => setSelectedProvider(mp.key)}
            >
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                <Image
                  src={mp.logo}
                  alt={mp.name}
                  width={24}
                  height={24}
                  className="h-6 w-6 object-contain"
                />
              </div>
              <div className="min-w-0">
                <p className="font-medium">{mp.name}</p>
                <p className="text-sm text-muted-foreground">{mp.description}</p>
              </div>
            </button>
          ))}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Anuluj
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ===================== WooCommerce Create Dialog =====================

function CreateWooCommerceListingDialog({
  product,
  onClose,
}: {
  product: Product;
  onClose: () => void;
}) {
  const { data: integrations } = useIntegrations();
  const wooIntegrationId = useMemo(
    () => integrations?.find((i) => i.provider === "woocommerce")?.id ?? "",
    [integrations]
  );

  const [price, setPrice] = useState(String(product.price));
  const [stock, setStock] = useState(String(product.stock_quantity));
  const [description, setDescription] = useState(
    product.description_long || product.description_short || ""
  );
  const [categories, setCategories] = useState("");

  const createListing = useCreateWooCommerceListing(product.id);

  const handleSubmit = () => {
    const cats = categories
      .split(",")
      .map((c) => c.trim())
      .filter(Boolean);

    createListing.mutate(
      {
        integration_id: wooIntegrationId,
        price_override: parseFloat(price) || undefined,
        stock_override: parseInt(stock) || undefined,
        description: description || undefined,
        categories: cats.length > 0 ? cats : undefined,
      },
      {
        onSuccess: () => {
          toast.success("Produkt wystawiony na WooCommerce");
          onClose();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Nie udalo sie wystawic produktu na WooCommerce"
          );
        },
      }
    );
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Wystaw na WooCommerce</DialogTitle>
          <DialogDescription>
            Publikacja produktu &quot;{product.name}&quot; w sklepie WooCommerce
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Price */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Cena (PLN)</Label>
              <Input
                type="number"
                step="0.01"
                min="0"
                value={price}
                onChange={(e) => setPrice(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Cena produktu: {product.price} PLN
              </p>
            </div>
            <div className="space-y-2">
              <Label>Stan magazynowy</Label>
              <Input
                type="number"
                min="0"
                value={stock}
                onChange={(e) => setStock(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Aktualny stan: {product.stock_quantity}
              </p>
            </div>
          </div>

          {/* Description */}
          <div className="space-y-2">
            <Label>Opis produktu</Label>
            <Textarea
              rows={4}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Opis produktu w sklepie WooCommerce"
            />
          </div>

          {/* Categories */}
          <div className="space-y-2">
            <Label>Kategorie WooCommerce</Label>
            <Input
              value={categories}
              onChange={(e) => setCategories(e.target.value)}
              placeholder="np. Czesci samochodowe, Oleje (oddzielone przecinkami)"
            />
            <p className="text-xs text-muted-foreground">
              Nazwy kategorii oddzielone przecinkami. Jesli kategoria nie istnieje, zostanie utworzona.
            </p>
          </div>

          {/* Product info summary */}
          <div className="rounded-md border bg-muted/50 p-3 space-y-1 text-sm">
            <p>
              <span className="text-muted-foreground">SKU/EAN:</span>{" "}
              {product.ean || product.sku || "---"}
            </p>
            <p>
              <span className="text-muted-foreground">Zdjecia:</span>{" "}
              {(product.images?.length ?? 0) > 0
                ? `${product.images!.length} zdjec`
                : "Brak"}
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Anuluj
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={createListing.isPending || !wooIntegrationId}
          >
            {createListing.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Wystaw na WooCommerce
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ===================== Allegro Create Dialog =====================

function CreateAllegroListingDialog({
  product,
  onClose,
}: {
  product: Product;
  onClose: () => void;
}) {
  const [step, setStep] = useState(1);

  // Integration ID from existing integrations
  const { data: integrations } = useIntegrations();
  const allegroIntegrationId = useMemo(
    () => integrations?.find((i) => i.provider === "allegro")?.id ?? "",
    [integrations]
  );

  // Step 1: Category
  const [parentCategoryId, setParentCategoryId] = useState<string | null>(null);
  const [categoryBreadcrumb, setCategoryBreadcrumb] = useState<
    { id: string; name: string }[]
  >([]);
  const [selectedCategoryId, setSelectedCategoryId] = useState("");
  const [selectedCategoryName, setSelectedCategoryName] = useState("");

  // Step 1: Category search
  const [categorySearchInput, setCategorySearchInput] = useState("");
  const [categorySearchQuery, setCategorySearchQuery] = useState("");
  useEffect(() => {
    const timer = setTimeout(() => setCategorySearchQuery(categorySearchInput.trim()), 400);
    return () => clearTimeout(timer);
  }, [categorySearchInput]);
  const { data: searchResults, isLoading: searchLoading } =
    useAllegroCategorySearch(categorySearchQuery);

  // Step 2: Parameters
  const [paramValues, setParamValues] = useState<
    Record<string, { valuesIds?: string[]; values?: string[] }>
  >({});

  // Step 3: Description
  const [descriptionHTML, setDescriptionHTML] = useState("");
  const [aiLoading, setAiLoading] = useState(false);

  // Step 4: Delivery/Policies
  const [shippingRateId, setShippingRateId] = useState("");
  const [returnPolicyId, setReturnPolicyId] = useState("");
  const [warrantyId, setWarrantyId] = useState("");
  const [handlingTime, setHandlingTime] = useState("PT24H");

  // Step 5: Price/Location
  const [priceOverride, setPriceOverride] = useState(String(product.price));
  const [stockOverride, setStockOverride] = useState(
    String(product.stock_quantity)
  );
  const [city, setCity] = useState("Warszawa");
  const [postCode, setPostCode] = useState("00-001");
  const [province, setProvince] = useState("MAZOWIECKIE");

  // Hooks
  const { data: categoriesData, isLoading: categoriesLoading, isError: categoriesError, error: categoriesErrorObj } =
    useAllegroCategories(parentCategoryId);
  const { data: paramsData, isLoading: paramsLoading } =
    useAllegroCategoryParams(selectedCategoryId || null);
  const createListing = useCreateProductListing(product.id);

  // Supplier link & saved Allegro parameter mappings
  const { data: supplierLink } = useProductSupplierLink(product.id);
  const { data: savedMappings } = useAllegroParameterMappings(
    supplierLink?.supplier_id ?? "",
    selectedCategoryId
  );

  // --- Auto-lookup: catalog → listing search → matching-categories ---
  const [catalogFallbackToName, setCatalogFallbackToName] = useState(false);
  const [catalogDone, setCatalogDone] = useState(false);
  const catalogSearchPhrase = useMemo(() => {
    if (!product.name) return "";
    const words = product.name.split(/[\s\-–—]+/).filter(Boolean);
    return words.slice(0, 5).join(" ");
  }, [product.name]);
  const catalogSearchParams = useMemo(() => {
    if (catalogDone) return undefined; // stop fetching once resolved
    if (product.ean && !catalogFallbackToName) return { phrase: product.ean, mode: "GTIN", limit: 1 };
    if (catalogSearchPhrase) return { phrase: catalogSearchPhrase, limit: 5 };
    return undefined;
  }, [product.ean, catalogSearchPhrase, catalogFallbackToName, catalogDone]);
  const { data: catalogResult, isError: catalogError, isLoading: catalogLoading } = useAllegroProductSearch(catalogSearchParams);
  const catalogProduct = catalogResult?.products?.[0];

  // Search marketplace offers by EAN (fires in parallel, used as parameter fallback)
  const { data: listingResult } = useAllegroListingSearch(product.ean || undefined);
  const listingOffer = useMemo(() => {
    if (!listingResult?.items) return undefined;
    return listingResult.items.regular?.[0] ?? listingResult.items.promoted?.[0];
  }, [listingResult]);

  // Fallback: EAN empty → try by name
  useEffect(() => {
    if (catalogFallbackToName || catalogDone || !product.ean) return;
    if (catalogError || (catalogResult && catalogResult.products.length === 0)) {
      setCatalogFallbackToName(true);
    }
  }, [catalogResult, catalogError, catalogFallbackToName, catalogDone, product.ean]);

  // Mark catalog done when name search also finishes
  useEffect(() => {
    if (catalogDone) return;
    if (catalogLoading) return;
    if (!catalogFallbackToName && product.ean) return; // still on EAN phase
    // Name search (or no-EAN search) finished
    if (catalogResult !== undefined || catalogError) {
      setCatalogDone(true);
    }
  }, [catalogResult, catalogError, catalogLoading, catalogFallbackToName, catalogDone, product.ean]);

  // Matching-categories fallback: fires when catalog AND listing found nothing
  const matchingQuery = useMemo(() => {
    if (!catalogDone || catalogProduct) return ""; // catalog found something, or not done yet
    if (listingOffer?.category?.id) return ""; // listing search found something
    return catalogSearchPhrase;
  }, [catalogDone, catalogProduct, listingOffer, catalogSearchPhrase]);
  const { data: matchingResult } = useAllegroCategorySearch(matchingQuery);
  const [autoApplied, setAutoApplied] = useState(false);

  // Auto-select category from catalog product → listing offer → matching-categories
  useEffect(() => {
    if (autoApplied || selectedCategoryId) return;

    // Option A: catalog product found — use its category + params
    if (catalogProduct?.category?.id) {
      setSelectedCategoryId(catalogProduct.category.id);
      setSelectedCategoryName(catalogProduct.name);
      setAutoApplied(true);
      setStep(2);
      toast.success("Znaleziono produkt w katalogu Allegro — kategoria i parametry uzupelnione automatycznie");
      return;
    }

    // Option B: listing search found an existing offer — use its category + params
    if (catalogDone && listingOffer?.category?.id) {
      setSelectedCategoryId(listingOffer.category.id);
      setSelectedCategoryName(listingOffer.name);
      setAutoApplied(true);
      setStep(2);
      toast.success("Znaleziono oferte na Allegro — kategoria i parametry pobrane automatycznie");
      return;
    }

    // Option C: matching-categories returned a suggestion
    if (catalogDone && matchingResult?.matchingCategories?.length) {
      const best = matchingResult.matchingCategories[0];
      setSelectedCategoryId(best.id);
      setSelectedCategoryName(best.name);
      setAutoApplied(true);
      setStep(2);
      toast.success(`Zasugerowano kategorie: ${best.name}`);
      return;
    }

    // Option D: everything tried, nothing found
    if (catalogDone && matchingResult !== undefined && !matchingResult?.matchingCategories?.length) {
      setAutoApplied(true); // give up, user picks manually
    }
  }, [catalogProduct, catalogDone, listingOffer, matchingResult, autoApplied, selectedCategoryId]);

  // Auto-fill parameters from product data when category params load
  useEffect(() => {
    if (!paramsData?.parameters?.length) return;

    const meta = product.metadata as Record<string, unknown> | undefined;
    const brand =
      (meta?.brand as string | undefined) ??
      (meta?.manufacturer as string | undefined) ??
      product.tags?.[0];

    // Supplier XML Specification/Attributes stored in metadata.attributes
    const supplierAttrs = (meta?.attributes ?? {}) as Record<string, string>;
    // Build lowercase lookup: "kolor" → "Czerwony"
    const supplierAttrsLower: Record<string, string> = {};
    for (const [key, val] of Object.entries(supplierAttrs)) {
      if (typeof val === "string" && val) {
        supplierAttrsLower[key.toLowerCase()] = val;
      }
    }

    const autoValues: Record<string, { valuesIds?: string[]; values?: string[] }> = {};

    // Alias map: supplier attribute names → Allegro parameter names
    const PARAM_ALIASES: Record<string, string[]> = {
      "marka": ["producent", "brand", "manufacturer", "firma"],
      "producent": ["marka", "brand", "manufacturer"],
      "kolor": ["kolor podstawowy", "color", "colour", "barwa"],
      "materiał": ["materiał wykonania", "tworzywo", "material"],
      "rozmiar": ["wymiar", "size", "wielkość"],
      "waga": ["masa", "weight", "gramatura"],
      "wzór": ["wzór dominujący", "deseń", "pattern"],
      "typ": ["rodzaj", "type", "kind"],
      "model": ["numer modelu", "model name"],
      "pojemność": ["pojemność [mah]", "capacity"],
      "szerokość": ["width"],
      "wysokość": ["height"],
      "długość": ["length", "głębokość"],
      "średnica": ["diameter"],
    };

    // Build reverse alias map: alias → canonical name
    const aliasToCanonical = new Map<string, string>();
    for (const [canonical, aliases] of Object.entries(PARAM_ALIASES)) {
      aliasToCanonical.set(canonical, canonical);
      for (const alias of aliases) {
        aliasToCanonical.set(alias, canonical);
      }
    }

    // Find supplier attribute value for a given Allegro parameter name
    const findSupplierAttrValue = (paramName: string): string | undefined => {
      const paramLower = paramName.toLowerCase();
      // Exact match
      if (supplierAttrsLower[paramLower]) return supplierAttrsLower[paramLower];
      // Alias match: check if param name or its aliases match any supplier attr
      const canonical = aliasToCanonical.get(paramLower);
      if (canonical) {
        // Check canonical name and all its aliases against supplier attrs
        if (supplierAttrsLower[canonical]) return supplierAttrsLower[canonical];
        for (const alias of PARAM_ALIASES[canonical] ?? []) {
          if (supplierAttrsLower[alias]) return supplierAttrsLower[alias];
        }
      }
      // Partial match: supplier attr key contains param name or vice versa
      for (const [attrKey, attrVal] of Object.entries(supplierAttrsLower)) {
        if (attrKey.includes(paramLower) || paramLower.includes(attrKey)) {
          return attrVal;
        }
      }
      return undefined;
    };

    const tryDictMatch = (
      param: AllegroCategoryParameter,
      text: string
    ): boolean => {
      if (param.type !== "dictionary" || !param.dictionary?.length) return false;
      const lower = text.toLowerCase().trim();
      // Exact match
      const exact = param.dictionary.find(
        (d) => d.value.toLowerCase() === lower
      );
      if (exact) {
        autoValues[param.id] = { valuesIds: [exact.id] };
        return true;
      }
      // Partial match: dict value contains our text or vice versa
      const partial = param.dictionary.find(
        (d) => {
          const dLower = d.value.toLowerCase();
          return dLower.includes(lower) || lower.includes(dLower);
        }
      );
      if (partial) {
        autoValues[param.id] = { valuesIds: [partial.id] };
        return true;
      }
      return false;
    };

    // Helper to resolve a product field value by name
    const getProductField = (fieldName: string): string | undefined => {
      switch (fieldName) {
        case "ean": return product.ean || undefined;
        case "sku": return product.sku || undefined;
        case "name": return product.name || undefined;
        case "brand": return brand || undefined;
        case "weight": return product.weight && product.weight > 0 ? String(product.weight) : undefined;
        case "price": return product.price ? String(product.price) : undefined;
        default: return undefined;
      }
    };

    // Build saved mappings lookup by param ID
    const savedMappingMap = new Map<string, (typeof savedMappings extends (infer T)[] | undefined ? T : never)>();
    if (savedMappings) {
      for (const m of savedMappings) {
        savedMappingMap.set(m.allegro_param_id, m);
      }
    }

    // Build catalog parameter lookup by ID (product catalog OR listing offer)
    const catalogParamMap = new Map<string, { values?: string[]; valuesIds?: string[] }>();
    const paramSource = catalogProduct?.parameters ?? listingOffer?.parameters;
    if (paramSource) {
      for (const cp of paramSource) {
        if (cp.valuesIds?.length) {
          catalogParamMap.set(cp.id, { valuesIds: cp.valuesIds });
        } else if (cp.values?.length) {
          catalogParamMap.set(cp.id, { values: cp.values });
        }
      }
    }

    for (const param of paramsData.parameters) {
      const nameLower = param.name.toLowerCase();

      // Priority 0: Apply parameters from Allegro catalog (most reliable)
      const catalogParam = catalogParamMap.get(param.id);
      if (catalogParam) {
        autoValues[param.id] = catalogParam;
        continue;
      }

      // Priority 1: Apply saved supplier mappings
      const mapping = savedMappingMap.get(param.id);
      if (mapping) {
        let value: string | undefined;
        switch (mapping.source_type) {
          case "attribute":
            value = supplierAttrsLower[mapping.source_key.toLowerCase()];
            break;
          case "field":
            value = getProductField(mapping.source_key);
            break;
          case "static":
            value = mapping.source_key;
            break;
        }
        if (value) {
          // Check value_mapping for dictionary translation
          const mappedDictId = mapping.value_mapping?.[value.toLowerCase()];
          if (mappedDictId) {
            autoValues[param.id] = { valuesIds: [mappedDictId] };
          } else if (param.type === "dictionary" && param.dictionary?.length) {
            if (!tryDictMatch(param, value)) {
              autoValues[param.id] = { values: [value] };
            }
          } else {
            autoValues[param.id] = { values: [value] };
          }
          continue;
        }
      }

      // Priority 2: Name-based auto-fill (existing logic)

      // EAN / GTIN
      if (
        (nameLower.includes("ean") || nameLower.includes("gtin")) &&
        product.ean
      ) {
        if (param.type === "dictionary" && param.dictionary?.length) {
          // Some categories have EAN as dictionary with custom values
          if (!tryDictMatch(param, product.ean)) {
            autoValues[param.id] = { values: [product.ean] };
          }
        } else {
          autoValues[param.id] = { values: [product.ean] };
        }
        continue;
      }

      // Marka / Producent (Brand/Manufacturer) — match in dictionary
      if (
        (nameLower === "marka" ||
          nameLower.includes("producent") ||
          nameLower === "brand" ||
          nameLower === "manufacturer") &&
        param.type === "dictionary" &&
        param.dictionary?.length &&
        brand
      ) {
        if (!tryDictMatch(param, brand)) {
          // Brand not in dictionary — set as free text if param allows it
          autoValues[param.id] = { values: [brand] };
        }
        continue;
      }

      // Stan (Condition) — auto-select "Nowy"
      if (
        nameLower === "stan" &&
        param.type === "dictionary" &&
        param.dictionary?.length
      ) {
        const match = param.dictionary.find(
          (d) => d.value.toLowerCase() === "nowy"
        );
        if (match) {
          autoValues[param.id] = { valuesIds: [match.id] };
        }
        continue;
      }

      // Numer katalogowy / Kod producenta (Catalog/part number) — fill from SKU
      if (
        (nameLower.includes("numer katalogowy") ||
          nameLower.includes("numer producenta") ||
          nameLower.includes("kod producenta") ||
          nameLower.includes("mpn")) &&
        product.sku
      ) {
        if (param.type === "dictionary" && param.dictionary?.length) {
          if (!tryDictMatch(param, product.sku)) {
            autoValues[param.id] = { values: [product.sku] };
          }
        } else {
          autoValues[param.id] = { values: [product.sku] };
        }
        continue;
      }

      // Waga (Weight)
      if (
        nameLower.includes("waga") &&
        product.weight &&
        product.weight > 0
      ) {
        if (param.type === "dictionary" && param.dictionary?.length) {
          const weightStr = String(product.weight);
          const match = param.dictionary.find((d) => d.value === weightStr);
          if (match) {
            autoValues[param.id] = { valuesIds: [match.id] };
          }
        } else {
          autoValues[param.id] = { values: [String(product.weight)] };
        }
        continue;
      }

      // Kod taryfy celnej (CN code) — fill from supplier cn_code attribute
      if (
        (nameLower.includes("taryf") || nameLower.includes("cn code")) &&
        supplierAttrsLower["cn_code"]
      ) {
        autoValues[param.id] = { values: [supplierAttrsLower["cn_code"]] };
        continue;
      }

      // Fallback: match supplier XML attributes with alias + partial matching
      const attrValue = findSupplierAttrValue(param.name);
      if (attrValue) {
        if (param.type === "dictionary" && param.dictionary?.length) {
          tryDictMatch(param, attrValue);
        } else {
          autoValues[param.id] = { values: [attrValue] };
        }
        continue;
      }

    }

    if (Object.keys(autoValues).length > 0) {
      setParamValues((prev) => {
        const merged = { ...autoValues };
        for (const [key, val] of Object.entries(prev)) {
          merged[key] = val;
        }
        return merged;
      });
    }
  }, [paramsData, product.ean, product.sku, product.weight, product.tags, product.metadata, savedMappings, catalogProduct, listingOffer]);

  // Category navigation
  const handleCategoryClick = (cat: AllegroCategory) => {
    if (cat.leaf) {
      setSelectedCategoryId(cat.id);
      setSelectedCategoryName(cat.name);
    } else {
      setParentCategoryId(cat.id);
      setCategoryBreadcrumb((prev) => [
        ...prev,
        { id: cat.id, name: cat.name },
      ]);
      setSelectedCategoryId("");
      setSelectedCategoryName("");
    }
  };

  const handleCategoryBreadcrumb = (index: number) => {
    if (index < 0) {
      setParentCategoryId(null);
      setCategoryBreadcrumb([]);
    } else {
      setParentCategoryId(categoryBreadcrumb[index].id);
      setCategoryBreadcrumb((prev) => prev.slice(0, index + 1));
    }
    setSelectedCategoryId("");
    setSelectedCategoryName("");
  };

  // Parameter change handler
  const handleParamChange = (
    paramId: string,
    type: string,
    value: string
  ) => {
    setParamValues((prev) => {
      const next = { ...prev };
      if (type === "dictionary") {
        next[paramId] = { valuesIds: [value] };
      } else {
        next[paramId] = { values: [value] };
      }
      return next;
    });
  };

  // Submit
  const handleSubmit = () => {
    const parameters = Object.entries(paramValues).map(([id, val]) => ({
      id,
      ...(val.valuesIds ? { valuesIds: val.valuesIds } : {}),
      ...(val.values ? { values: val.values } : {}),
    }));

    createListing.mutate(
      {
        integration_id: allegroIntegrationId,
        category_id: selectedCategoryId,
        parameters,
        description_html: descriptionHTML || undefined,
        shipping_rate_id: shippingRateId,
        return_policy_id: returnPolicyId,
        warranty_id: warrantyId,
        handling_time: handlingTime,
        price_override: parseFloat(priceOverride) || undefined,
        stock_override: parseInt(stockOverride) || undefined,
        location: { city, province, post_code: postCode, country_code: "PL" },
      },
      {
        onSuccess: () => {
          toast.success("Oferta zostala wystawiona na Allegro");
          onClose();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Nie udalo sie wystawic oferty"
          );
        },
      }
    );
  };

  const canProceedStep1 = !!selectedCategoryId;

  // Validate all required parameters have values
  const missingRequiredParams = useMemo(() => {
    if (!paramsData?.parameters?.length) return [];
    return paramsData.parameters.filter((p) => {
      if (!p.required) return false;
      const val = paramValues[p.id];
      if (!val) return true;
      const hasValues = val.values?.some((v) => v.trim() !== "");
      const hasIds = val.valuesIds?.some((v) => v.trim() !== "");
      return !hasValues && !hasIds;
    });
  }, [paramsData, paramValues]);
  const canProceedStep2 = missingRequiredParams.length === 0;
  const canProceedStep4 =
    !!shippingRateId && !!returnPolicyId && !!warrantyId;

  // Initialize description when entering step 3
  useEffect(() => {
    if (step === 3 && !descriptionHTML && product) {
      const text = product.description_long || product.description_short || "";
      setDescriptionHTML(plainTextToHTML(text));
    }
  }, [step]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Wystaw na Allegro — krok {step} z 5</DialogTitle>
          <DialogDescription>
            {step === 1 && "Wybierz kategorie Allegro dla produktu"}
            {step === 2 && "Wypelnij parametry wymagane przez kategorie"}
            {step === 3 && "Edytuj opis oferty"}
            {step === 4 &&
              "Wybierz ustawienia dostawy i polityki sprzedazy"}
            {step === 5 &&
              "Ustaw cene, stan magazynowy i lokalizacje"}
          </DialogDescription>
        </DialogHeader>

        {/* Step 1: Category */}
        {step === 1 && (
          <div className="space-y-4">
            {/* Search input */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Wyszukaj kategorie, np. olej silnikowy..."
                value={categorySearchInput}
                onChange={(e) => setCategorySearchInput(e.target.value)}
                className="pl-9"
              />
            </div>

            {/* Search results */}
            {categorySearchQuery.length >= 2 ? (
              <div className="space-y-1">
                {searchLoading ? (
                  <div className="space-y-2">
                    {Array.from({ length: 4 }).map((_, i) => (
                      <Skeleton key={i} className="h-12 w-full" />
                    ))}
                  </div>
                ) : !searchResults?.matchingCategories?.length ? (
                  <p className="py-4 text-center text-muted-foreground text-sm">
                    Brak pasujacych kategorii dla &quot;{categorySearchQuery}&quot;
                  </p>
                ) : (
                  <div className="space-y-1">
                    {searchResults.matchingCategories.map((cat) => {
                      const path = buildCategoryPath(cat);
                      return (
                        <button
                          key={cat.id}
                          onClick={() => {
                            setSelectedCategoryId(cat.id);
                            setSelectedCategoryName(cat.name);
                            setCategorySearchInput("");
                            setCategorySearchQuery("");
                          }}
                          className={`flex items-start gap-3 w-full rounded-md border p-3 text-left text-sm transition-colors hover:bg-muted/50 ${
                            selectedCategoryId === cat.id
                              ? "border-primary bg-primary/5"
                              : "border-transparent"
                          }`}
                        >
                          <Tag className="h-4 w-4 shrink-0 text-muted-foreground mt-0.5" />
                          <div className="flex-1 min-w-0">
                            <span className="font-medium">{cat.name}</span>
                            <p className="text-xs text-muted-foreground truncate">
                              {path}
                            </p>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            ) : (
              <>
                {/* Breadcrumb */}
                <div className="flex flex-wrap items-center gap-1 text-sm">
                  <button
                    onClick={() => handleCategoryBreadcrumb(-1)}
                    className="text-primary hover:underline font-medium"
                  >
                    Wszystkie kategorie
                  </button>
                  {categoryBreadcrumb.map((item, idx) => (
                    <span key={item.id} className="flex items-center gap-1">
                      <ChevronRight className="h-3 w-3 text-muted-foreground" />
                      <button
                        onClick={() => handleCategoryBreadcrumb(idx)}
                        className={
                          idx === categoryBreadcrumb.length - 1
                            ? "font-medium"
                            : "text-primary hover:underline"
                        }
                      >
                        {item.name}
                      </button>
                    </span>
                  ))}
                </div>

                {/* Category grid */}
                {categoriesLoading ? (
                  <div className="space-y-2">
                    {Array.from({ length: 6 }).map((_, i) => (
                      <Skeleton key={i} className="h-10 w-full" />
                    ))}
                  </div>
                ) : categoriesError ? (
                  <div className="rounded-md border border-destructive/50 bg-destructive/5 p-4 text-center space-y-2">
                    <p className="text-sm text-destructive font-medium">
                      Nie udalo sie pobrac kategorii Allegro
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {categoriesErrorObj instanceof Error
                        ? categoriesErrorObj.message
                        : "Sprawdz czy integracja Allegro jest skonfigurowana i autoryzowana."}
                    </p>
                    <Button
                      variant="outline"
                      size="sm"
                      asChild
                    >
                      <Link href="/marketplaces/allegro">
                        Przejdz do ustawien Allegro
                      </Link>
                    </Button>
                  </div>
                ) : !categoriesData?.categories?.length ? (
                  <p className="py-4 text-center text-muted-foreground">
                    Brak kategorii do wyswietlenia
                  </p>
                ) : (
                  <div className="grid grid-cols-1 gap-1 sm:grid-cols-2">
                    {categoriesData.categories.map((cat) => (
                      <button
                        key={cat.id}
                        onClick={() => handleCategoryClick(cat)}
                        className={`flex items-center gap-3 rounded-md border p-3 text-left text-sm transition-colors hover:bg-muted/50 ${
                          selectedCategoryId === cat.id
                            ? "border-primary bg-primary/5"
                            : "border-transparent"
                        }`}
                      >
                        {cat.leaf ? (
                          <Tag className="h-4 w-4 shrink-0 text-muted-foreground" />
                        ) : (
                          <FolderOpen className="h-4 w-4 shrink-0 text-muted-foreground" />
                        )}
                        <span className="flex-1 truncate">{cat.name}</span>
                        {cat.leaf ? (
                          <Badge
                            variant="secondary"
                            className="text-xs shrink-0"
                          >
                            Lisc
                          </Badge>
                        ) : (
                          <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
                        )}
                      </button>
                    ))}
              </div>
            )}
              </>
            )}

            {selectedCategoryId && (
              <div className="rounded-md border border-primary/30 bg-primary/5 p-3">
                <p className="text-sm">
                  <span className="font-medium">Wybrana kategoria:</span>{" "}
                  {selectedCategoryName}{" "}
                  <span className="text-muted-foreground">
                    (ID: {selectedCategoryId})
                  </span>
                </p>
              </div>
            )}
          </div>
        )}

        {/* Step 2: Parameters */}
        {step === 2 && (
          <div className="space-y-4">
            {paramsLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
            ) : !paramsData?.parameters?.length ? (
              <p className="py-4 text-center text-muted-foreground">
                Brak parametrow dla tej kategorii. Mozesz przejsc dalej.
              </p>
            ) : (
              <div className="space-y-4">
                <p className="text-sm text-muted-foreground">
                  Wypelnij parametry oznaczone * (wymagane). Pozostale sa
                  opcjonalne.
                </p>
                {paramsData.parameters.map((param) => (
                  <ParameterField
                    key={param.id}
                    param={param}
                    value={paramValues[param.id]}
                    onChange={(value) =>
                      handleParamChange(param.id, param.type, value)
                    }
                  />
                ))}
                {missingRequiredParams.length > 0 && (
                  <div className="rounded-md border border-destructive/50 bg-destructive/5 p-3 text-sm">
                    <p className="font-medium text-destructive mb-1">
                      Brakuje wymaganych parametrow ({missingRequiredParams.length}):
                    </p>
                    <ul className="list-disc list-inside text-muted-foreground">
                      {missingRequiredParams.map((p) => (
                        <li key={p.id}>{p.name}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* Step 3: Description */}
        {step === 3 && (
          <div className="space-y-4">
            <div>
              <h3 className="text-lg font-medium">Opis oferty</h3>
              <p className="text-sm text-muted-foreground">
                Edytuj opis przed wystawieniem. Dozwolone: naglowki, paragrafy, listy.
              </p>
            </div>
            <DescriptionEditor
              value={descriptionHTML}
              onChange={setDescriptionHTML}
              placeholder="Wpisz opis oferty..."
              onAiGenerate={async () => {
                setAiLoading(true);
                try {
                  const res = await apiClient<AISuggestion>("/v1/ai/describe", {
                    method: "POST",
                    body: JSON.stringify({
                      product_id: product.id,
                      marketplace: "allegro",
                      format: "html",
                    }),
                  });
                  if (res.long_description) {
                    setDescriptionHTML(res.long_description);
                  }
                } finally {
                  setAiLoading(false);
                }
              }}
              onAiImprove={async (html: string) => {
                setAiLoading(true);
                try {
                  const res = await apiClient<AITextResult>("/v1/ai/improve", {
                    method: "POST",
                    body: JSON.stringify({
                      description: html,
                      format: "html",
                    }),
                  });
                  if (res.description) {
                    setDescriptionHTML(res.description);
                  }
                } finally {
                  setAiLoading(false);
                }
              }}
              aiLoading={aiLoading}
            />
          </div>
        )}

        {/* Step 4: Delivery & Policies */}
        {step === 4 && (
          <Step3DeliveryPolicies
            shippingRateId={shippingRateId}
            setShippingRateId={setShippingRateId}
            returnPolicyId={returnPolicyId}
            setReturnPolicyId={setReturnPolicyId}
            warrantyId={warrantyId}
            setWarrantyId={setWarrantyId}
            handlingTime={handlingTime}
            setHandlingTime={setHandlingTime}
            product={product}
          />
        )}

        {/* Step 5: Price & Location */}
        {step === 5 && (
          <div className="space-y-6">
            {/* Summary */}
            <div className="rounded-md border bg-muted/50 p-4 space-y-2">
              <p className="text-sm font-medium">Podsumowanie</p>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div>
                  <span className="text-muted-foreground">Produkt:</span>{" "}
                  {product.name}
                </div>
                <div>
                  <span className="text-muted-foreground">SKU:</span>{" "}
                  {product.sku || "---"}
                </div>
                <div>
                  <span className="text-muted-foreground">Kategoria:</span>{" "}
                  {selectedCategoryName}
                </div>
                <div>
                  <span className="text-muted-foreground">
                    Czas realizacji:
                  </span>{" "}
                  {HANDLING_TIME_OPTIONS.find(
                    (o) => o.value === handlingTime
                  )?.label ?? handlingTime}
                </div>
              </div>
            </div>

            <Separator />

            {/* Price & Stock */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Cena (PLN)</Label>
                <Input
                  type="number"
                  step="0.01"
                  min="0"
                  value={priceOverride}
                  onChange={(e) => setPriceOverride(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  Cena produktu: {product.price} PLN
                </p>
              </div>
              <div className="space-y-2">
                <Label>Stan magazynowy</Label>
                <Input
                  type="number"
                  min="0"
                  value={stockOverride}
                  onChange={(e) => setStockOverride(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  Aktualny stan: {product.stock_quantity}
                </p>
              </div>
            </div>

            <Separator />

            {/* Location */}
            <div className="space-y-4">
              <p className="text-sm font-medium">Lokalizacja przedmiotu</p>
              <div className="grid grid-cols-3 gap-4">
                <div className="space-y-2">
                  <Label>Miasto</Label>
                  <Input
                    value={city}
                    onChange={(e) => setCity(e.target.value)}
                    placeholder="np. Warszawa"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Kod pocztowy</Label>
                  <Input
                    value={postCode}
                    onChange={(e) => setPostCode(e.target.value)}
                    placeholder="00-001"
                    maxLength={6}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Wojewodztwo</Label>
                  <Select value={province} onValueChange={setProvince}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {PROVINCES.map((prov) => (
                        <SelectItem key={prov} value={prov}>
                          {PROVINCE_LABELS[prov]}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>
          </div>
        )}

        <DialogFooter className="flex justify-between sm:justify-between">
          <div>
            {step > 1 && (
              <Button
                variant="outline"
                onClick={() => setStep((s) => s - 1)}
              >
                Wstecz
              </Button>
            )}
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={onClose}>
              Anuluj
            </Button>
            {step < 5 ? (
              <Button
                onClick={() => setStep((s) => s + 1)}
                disabled={
                  step === 1
                    ? !canProceedStep1
                    : step === 2
                      ? !canProceedStep2
                      : step === 4
                        ? !canProceedStep4
                        : false
                }
              >
                Dalej
              </Button>
            ) : (
              <Button
                onClick={handleSubmit}
                disabled={createListing.isPending}
              >
                {createListing.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Wystaw oferte
              </Button>
            )}
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ===================== Step 3: Delivery & Policies =====================

function Step3DeliveryPolicies({
  shippingRateId,
  setShippingRateId,
  returnPolicyId,
  setReturnPolicyId,
  warrantyId,
  setWarrantyId,
  handlingTime,
  setHandlingTime,
  product,
}: {
  shippingRateId: string;
  setShippingRateId: (v: string) => void;
  returnPolicyId: string;
  setReturnPolicyId: (v: string) => void;
  warrantyId: string;
  setWarrantyId: (v: string) => void;
  handlingTime: string;
  setHandlingTime: (v: string) => void;
  product: Product;
}) {
  const { data: shippingRatesData } = useAllegroShippingRates();
  const { data: returnPoliciesData } = useAllegroReturnPolicies();
  const { data: warrantiesData } = useAllegroWarranties();
  const createReturnPolicy = useCreateAllegroReturnPolicy();
  const createWarranty = useCreateAllegroWarranty();
  const autoGenerate = useAutoGenerateShippingRate();

  const handleCreateDefaultReturnPolicy = () => {
    createReturnPolicy.mutate(
      {
        name: "Standardowa polityka zwrotow",
        availability: { range: "FULL" },
        withdrawalPeriod: "P14D",
        returnCost: { coveredBy: "BUYER" },
        options: {
          cashOnDeliveryNotAllowed: false,
          freeAccessoriesReturnRequired: false,
          refundLoweredByReceivedDiscount: false,
          businessReturnAllowed: false,
          collectBySellerOnly: false,
        },
        address: { name: "Firma", street: "Adres", city: "Miasto", postCode: "00-000", countryCode: "PL" },
        contact: { email: "email@firma.pl", phoneNumber: "123456789" },
      },
      {
        onSuccess: (data) => {
          toast.success("Utworzono domyslna polityke zwrotow");
          if (data?.id) setReturnPolicyId(data.id);
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Nie udalo sie utworzyc polityki zwrotow"
          );
        },
      }
    );
  };

  const handleCreateDefaultWarranty = () => {
    createWarranty.mutate(
      {
        name: "Rekojmia ustawowa",
        individual: { period: "P2Y", type: "IMPLIED_WARRANTY" },
        corporate: { period: "P1Y", type: "IMPLIED_WARRANTY" },
      },
      {
        onSuccess: (data) => {
          toast.success("Utworzono domyslna rekojmie");
          if (data?.id) setWarrantyId(data.id);
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Nie udalo sie utworzyc rekojmi"
          );
        },
      }
    );
  };

  return (
    <div className="space-y-6">
      {/* Shipping rate */}
      <div className="space-y-2">
        <Label>
          Cennik wysylki <span className="text-destructive">*</span>
        </Label>
        {shippingRatesData?.shippingRates?.length ? (
          <Select value={shippingRateId} onValueChange={setShippingRateId}>
            <SelectTrigger>
              <SelectValue placeholder="Wybierz cennik wysylki" />
            </SelectTrigger>
            <SelectContent>
              {shippingRatesData.shippingRates.map((rate) => (
                <SelectItem key={rate.id} value={rate.id}>
                  {rate.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <p className="text-sm text-muted-foreground">
            Brak cennikow wysylki.
          </p>
        )}
        <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
          <span>Nie masz cennika?</span>
          {product.weight && product.weight > 0 ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                autoGenerate.mutate(
                  {
                    weight_kg: product.weight ?? 0,
                    width_cm: product.width ?? 0,
                    height_cm: product.height ?? 0,
                    length_cm: product.depth ?? 0,
                    name: `InPost - ${product.name}`.slice(0, 64),
                  },
                  {
                    onSuccess: (data) => {
                      toast.success("Wygenerowano cennik InPost");
                      if (data?.id) setShippingRateId(data.id);
                    },
                    onError: (error) => {
                      toast.error(
                        error instanceof Error
                          ? error.message
                          : "Nie udalo sie wygenerowac cennika"
                      );
                    },
                  }
                );
              }}
              disabled={autoGenerate.isPending}
            >
              {autoGenerate.isPending && (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              )}
              Wygeneruj z InPost
            </Button>
          ) : null}
          <Link
            href="/marketplaces/allegro/delivery"
            className="text-primary hover:underline"
            target="_blank"
          >
            Utworz recznie
          </Link>
        </div>
      </div>

      <Separator />

      {/* Return policy */}
      <div className="space-y-2">
        <Label>
          Polityka zwrotow <span className="text-destructive">*</span>
        </Label>
        {returnPoliciesData?.returnPolicies?.length ? (
          <Select value={returnPolicyId} onValueChange={setReturnPolicyId}>
            <SelectTrigger>
              <SelectValue placeholder="Wybierz polityke zwrotow" />
            </SelectTrigger>
            <SelectContent>
              {returnPoliciesData.returnPolicies.map((policy) => (
                <SelectItem key={policy.id} value={policy.id}>
                  {policy.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">
              Brak polityk zwrotow.
            </p>
            <Button
              variant="outline"
              size="sm"
              onClick={handleCreateDefaultReturnPolicy}
              disabled={createReturnPolicy.isPending}
            >
              {createReturnPolicy.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Utworz domyslna polityke
            </Button>
          </div>
        )}
        <p className="text-xs text-muted-foreground">
          Nie masz polityki?{" "}
          <Link
            href="/marketplaces/allegro/policies"
            className="text-primary hover:underline"
            target="_blank"
          >
            Utworz nowa polityke zwrotow
          </Link>
        </p>
      </div>

      <Separator />

      {/* Warranty */}
      <div className="space-y-2">
        <Label>
          Rekojmia <span className="text-destructive">*</span>
        </Label>
        {warrantiesData?.impliedWarranties?.length ? (
          <Select value={warrantyId} onValueChange={setWarrantyId}>
            <SelectTrigger>
              <SelectValue placeholder="Wybierz rekojmie" />
            </SelectTrigger>
            <SelectContent>
              {warrantiesData.impliedWarranties.map((warranty) => (
                <SelectItem key={warranty.id} value={warranty.id}>
                  {warranty.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">
              Brak rekojmi.
            </p>
            <Button
              variant="outline"
              size="sm"
              onClick={handleCreateDefaultWarranty}
              disabled={createWarranty.isPending}
            >
              {createWarranty.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Utworz domyslna rekojmie
            </Button>
          </div>
        )}
        <p className="text-xs text-muted-foreground">
          Nie masz rekojmi?{" "}
          <Link
            href="/marketplaces/allegro/policies"
            className="text-primary hover:underline"
            target="_blank"
          >
            Utworz nowa rekojmie
          </Link>
        </p>
      </div>

      <Separator />

      {/* Handling time */}
      <div className="space-y-2">
        <Label>Czas realizacji</Label>
        <Select value={handlingTime} onValueChange={setHandlingTime}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {HANDLING_TIME_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}

// ===================== Helpers =====================

/** Builds a readable path string from nested parent chain, e.g. "Motoryzacja > Oleje > Oleje silnikowe" */
function buildCategoryPath(cat: AllegroMatchingCategory): string {
  const parts: string[] = [];
  let current: AllegroMatchingCategory | null | undefined = cat.parent;
  while (current) {
    parts.unshift(current.name);
    current = current.parent;
  }
  return parts.length > 0 ? parts.join(" > ") : "";
}

// ===================== Parameter Field =====================

const LARGE_DICT_THRESHOLD = 50;
const MAX_VISIBLE_ITEMS = 50;

function ParameterField({
  param,
  value,
  onChange,
}: {
  param: AllegroCategoryParameter;
  value?: { valuesIds?: string[]; values?: string[] };
  onChange: (value: string) => void;
}) {
  const currentValue =
    value?.valuesIds?.[0] ?? value?.values?.[0] ?? "";

  if (
    param.type === "dictionary" &&
    param.dictionary &&
    param.dictionary.length > 0
  ) {
    // Large dictionaries get a searchable combobox
    if (param.dictionary.length > LARGE_DICT_THRESHOLD) {
      return (
        <DictionaryCombobox
          param={param}
          currentValue={currentValue}
          onChange={onChange}
        />
      );
    }

    // Small dictionaries keep the simple Select
    return (
      <div className="space-y-2">
        <Label>
          {param.name}
          {param.required && (
            <span className="text-destructive"> *</span>
          )}
          {param.unit && (
            <span className="text-muted-foreground ml-1">
              ({param.unit})
            </span>
          )}
        </Label>
        <Select value={currentValue} onValueChange={onChange}>
          <SelectTrigger>
            <SelectValue placeholder={`Wybierz ${param.name.toLowerCase()}`} />
          </SelectTrigger>
          <SelectContent>
            {param.dictionary.map((d) => (
              <SelectItem key={d.id} value={d.id}>
                {d.value}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <Label>
        {param.name}
        {param.required && (
          <span className="text-destructive"> *</span>
        )}
        {param.unit && (
          <span className="text-muted-foreground ml-1">
            ({param.unit})
          </span>
        )}
      </Label>
      <Input
        type={
          param.type === "integer" || param.type === "float"
            ? "number"
            : "text"
        }
        step={param.type === "float" ? "0.01" : undefined}
        min={param.restrictions?.min}
        max={param.restrictions?.max}
        value={currentValue}
        onChange={(e) => onChange(e.target.value)}
        placeholder={`Wpisz ${param.name.toLowerCase()}`}
      />
    </div>
  );
}

// ===================== Dictionary Combobox =====================

function DictionaryCombobox({
  param,
  currentValue,
  onChange,
}: {
  param: AllegroCategoryParameter;
  currentValue: string;
  onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");

  const selectedLabel = useMemo(() => {
    if (!currentValue || !param.dictionary) return "";
    return param.dictionary.find((d) => d.id === currentValue)?.value ?? "";
  }, [currentValue, param.dictionary]);

  // Client-side filter: only render top N matches
  const filteredItems = useMemo(() => {
    if (!param.dictionary) return [];
    if (!search.trim()) return param.dictionary.slice(0, MAX_VISIBLE_ITEMS);
    const q = search.toLowerCase();
    const matches: typeof param.dictionary = [];
    for (const d of param.dictionary) {
      if (d.value.toLowerCase().includes(q)) {
        matches.push(d);
        if (matches.length >= MAX_VISIBLE_ITEMS) break;
      }
    }
    return matches;
  }, [param.dictionary, search]);

  const totalCount = param.dictionary?.length ?? 0;

  return (
    <div className="space-y-2">
      <Label>
        {param.name}
        {param.required && (
          <span className="text-destructive"> *</span>
        )}
        {param.unit && (
          <span className="text-muted-foreground ml-1">
            ({param.unit})
          </span>
        )}
      </Label>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className="w-full justify-between font-normal"
          >
            <span className="truncate">
              {selectedLabel || `Wybierz ${param.name.toLowerCase()}`}
            </span>
            <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-[--radix-popover-trigger-width] p-0" align="start">
          <Command shouldFilter={false}>
            <CommandInput
              placeholder={`Szukaj (${totalCount} opcji)...`}
              value={search}
              onValueChange={setSearch}
            />
            <CommandList>
              <CommandEmpty>Brak wynikow</CommandEmpty>
              <CommandGroup>
                {filteredItems.map((d) => (
                  <CommandItem
                    key={d.id}
                    value={d.id}
                    onSelect={() => {
                      onChange(d.id);
                      setOpen(false);
                      setSearch("");
                    }}
                  >
                    <Check
                      className={`mr-2 h-4 w-4 ${
                        currentValue === d.id ? "opacity-100" : "opacity-0"
                      }`}
                    />
                    {d.value}
                  </CommandItem>
                ))}
                {!search.trim() && totalCount > MAX_VISIBLE_ITEMS && (
                  <p className="py-2 px-3 text-xs text-muted-foreground text-center">
                    Wpisz tekst, aby wyszukac wsrod {totalCount} opcji
                  </p>
                )}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}
