"use client";

import { useState, useMemo, useEffect, useCallback } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import {
  ArrowLeft,
  ChevronRight,
  ExternalLink,
  Folder,
  FolderOpen,
  Info,
  Loader2,
  Package,
  Plus,
  RotateCcw,
  Search,
  Tag,
  Trash2,
} from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { EmptyState } from "@/components/shared/empty-state";
import { useProduct } from "@/hooks/use-products";
import { useIntegrations } from "@/hooks/use-integrations";
import {
  useProductListings,
  useCreateProductListing,
  useDeleteProductListing,
  useSyncProductListing,
  useAllegroCategories,
  useAllegroCategorySearch,
  useAllegroCategoryParams,
  useAllegroShippingRates,
  useAllegroReturnPolicies,
  useAllegroWarranties,
  useCreateAllegroReturnPolicy,
  useCreateAllegroWarranty,
  useAutoGenerateShippingRate,
} from "@/hooks/use-allegro";
import type {
  AllegroCategory,
  AllegroCategoryParameter,
  AllegroMatchingCategory,
  ProductListing,
  CreateProductListingRequest,
} from "@/hooks/use-allegro";
import type { Product } from "@/types/api";
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
import { Check, ChevronsUpDown } from "lucide-react";

// ===================== Constants =====================

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
  const [showCreate, setShowCreate] = useState(false);
  const deleteListing = useDeleteProductListing(params.id);
  const syncListing = useSyncProductListing(params.id);

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
            Wystaw na Allegro
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
            <CardHeader>
              <CardTitle>Aktywne oferty ({listings.length})</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Platforma</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>ID oferty</TableHead>
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
                      onSync={(id) =>
                        syncListing.mutate(id, {
                          onSuccess: () =>
                            toast.success("Zsynchronizowano"),
                          onError: () =>
                            toast.error("Blad synchronizacji"),
                        })
                      }
                      onDelete={(id) => {
                        if (confirm("Usunac oferte?")) {
                          deleteListing.mutate(id, {
                            onSuccess: () =>
                              toast.success("Oferta usunieta"),
                            onError: () =>
                              toast.error("Blad usuwania"),
                          });
                        }
                      }}
                    />
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        )}

        {/* Create dialog */}
        {showCreate && product && (
          <CreateAllegroListingDialog
            product={product}
            onClose={() => setShowCreate(false)}
          />
        )}
      </div>
    </AdminGuard>
  );
}

// ===================== Listing Row =====================

function ListingRow({
  listing,
  onSync,
  onDelete,
}: {
  listing: ProductListing;
  onSync: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  return (
    <TableRow>
      <TableCell>
        <Badge variant="outline">Allegro</Badge>
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
          {listing.external_id && (
            <Button variant="ghost" size="sm" asChild>
              <a
                href={`https://allegro.pl/oferta/${listing.external_id}`}
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
            onClick={() => onDelete(listing.id)}
          >
            <Trash2 className="h-4 w-4 text-destructive" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

// ===================== Create Dialog =====================

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

  // Step 3: Delivery/Policies
  const [shippingRateId, setShippingRateId] = useState("");
  const [returnPolicyId, setReturnPolicyId] = useState("");
  const [warrantyId, setWarrantyId] = useState("");
  const [handlingTime, setHandlingTime] = useState("PT24H");

  // Step 4: Price/Location
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
  const canProceedStep2 = true;
  const canProceedStep3 =
    !!shippingRateId && !!returnPolicyId && !!warrantyId;

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Wystaw na Allegro — krok {step} z 4</DialogTitle>
          <DialogDescription>
            {step === 1 && "Wybierz kategorie Allegro dla produktu"}
            {step === 2 && "Wypelnij parametry wymagane przez kategorie"}
            {step === 3 &&
              "Wybierz ustawienia dostawy i polityki sprzedazy"}
            {step === 4 &&
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
                      <Link href="/integrations/allegro">
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
              </div>
            )}
          </div>
        )}

        {/* Step 3: Delivery & Policies */}
        {step === 3 && (
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

        {/* Step 4: Price & Location */}
        {step === 4 && (
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
            {step < 4 ? (
              <Button
                onClick={() => setStep((s) => s + 1)}
                disabled={
                  step === 1
                    ? !canProceedStep1
                    : step === 2
                      ? !canProceedStep2
                      : !canProceedStep3
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
            href="/integrations/allegro/delivery"
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
            href="/integrations/allegro/policies"
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
            href="/integrations/allegro/policies"
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
