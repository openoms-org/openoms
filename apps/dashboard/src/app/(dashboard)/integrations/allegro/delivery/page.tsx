"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  Plus,
  Truck,
  Settings,
  ChevronDown,
  ChevronUp,
  Info,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAllegroDeliverySettings,
  useUpdateAllegroDeliverySettings,
  useAllegroShippingRates,
  useCreateAllegroShippingRate,
  useAllegroDeliveryMethods,
} from "@/hooks/use-allegro";
import type {
  AllegroDeliverySettings,
  AllegroShippingRateSet,
  AllegroCreateShippingRateRequest,
  AllegroDeliveryMethodItem,
} from "@/hooks/use-allegro";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
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
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";

export default function AllegroDeliveryPage() {
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
            <h1 className="text-2xl font-bold">Dostawa Allegro</h1>
            <p className="text-muted-foreground">
              Ustawienia dostawy i cenniki wysylki
            </p>
          </div>
        </div>

        {/* Help section */}
        <div className="rounded-lg border bg-muted/50 p-4 flex gap-3">
          <Info className="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div className="space-y-1 text-sm">
            <p className="font-medium">Ustawienia dostawy — co tu skonfigurujesz?</p>
            <ul className="list-disc list-inside space-y-0.5 text-muted-foreground">
              <li><strong>Prog darmowej dostawy</strong> — kwota zamowienia, od ktorej kupujacy nie placi za wysylke.</li>
              <li><strong>Polityka laczenia kosztow</strong> — jak liczone sa koszty dostawy przy wielu produktach: MIN (najtansza), MAX (najdrozsza) lub SUM (suma).</li>
              <li><strong>Dostawa zagraniczna</strong> — wlacz, jesli wysylasz za granice.</li>
              <li><strong>Cenniki wysylki</strong> — tworzysz cenniki z cenami za pierwsza i kolejna sztuke dla kazdej metody dostawy. Cennik przypisujesz do ofert.</li>
              <li><strong>Metody dostawy</strong> — lista dostepnych metod dostawy na Allegro (InPost, DPD, Poczta Polska itd.) z ich ID do uzycia w cennikach.</li>
            </ul>
          </div>
        </div>

        <Tabs defaultValue="settings">
          <TabsList>
            <TabsTrigger value="settings">
              <Settings className="mr-2 h-4 w-4" />
              Ustawienia dostawy
            </TabsTrigger>
            <TabsTrigger value="rates">
              <Truck className="mr-2 h-4 w-4" />
              Cenniki wysylki
            </TabsTrigger>
          </TabsList>

          <TabsContent value="settings" className="space-y-6 mt-4">
            <DeliverySettingsSection />
            <DeliveryMethodsSection />
          </TabsContent>

          <TabsContent value="rates" className="mt-4">
            <ShippingRatesSection />
          </TabsContent>
        </Tabs>
      </div>
    </AdminGuard>
  );
}

function DeliverySettingsSection() {
  const { data, isLoading } = useAllegroDeliverySettings();
  const updateSettings = useUpdateAllegroDeliverySettings();

  const [freeDeliveryThreshold, setFreeDeliveryThreshold] = useState("");
  const [joinPolicy, setJoinPolicy] = useState("MIN");
  const [abroadEnabled, setAbroadEnabled] = useState(false);
  const [customCostAllowed, setCustomCostAllowed] = useState(false);
  const [initialized, setInitialized] = useState(false);

  // Initialize form values from server data
  if (data && !initialized) {
    setFreeDeliveryThreshold(
      data.freeDelivery?.threshold?.amount ?? ""
    );
    setJoinPolicy(data.joinPolicy?.strategy ?? "MIN");
    setAbroadEnabled(data.abroadDelivery?.enabled ?? false);
    setCustomCostAllowed(data.customCost?.allowed ?? false);
    setInitialized(true);
  }

  const handleSave = () => {
    const settings: AllegroDeliverySettings = {
      joinPolicy: { strategy: joinPolicy },
      abroadDelivery: { enabled: abroadEnabled },
      customCost: { allowed: customCostAllowed },
    };

    if (freeDeliveryThreshold) {
      settings.freeDelivery = {
        amount: { amount: freeDeliveryThreshold, currency: "PLN" },
        threshold: { amount: freeDeliveryThreshold, currency: "PLN" },
      };
    }

    updateSettings.mutate(settings, {
      onSuccess: () => toast.success("Ustawienia dostawy zaktualizowane"),
      onError: () =>
        toast.error("Nie udalo sie zaktualizowac ustawien dostawy"),
    });
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Ustawienia dostawy</CardTitle>
        <CardDescription>
          Konfiguracja ogolnych ustawien dostawy na Allegro
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-6 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="free-delivery-threshold">
              Prog darmowej dostawy (PLN)
            </Label>
            <Input
              id="free-delivery-threshold"
              type="number"
              step="0.01"
              min="0"
              value={freeDeliveryThreshold}
              onChange={(e) => setFreeDeliveryThreshold(e.target.value)}
              placeholder="np. 100.00"
            />
          </div>

          <div className="space-y-2">
            <Label>Polityka laczenia kosztow</Label>
            <Select value={joinPolicy} onValueChange={setJoinPolicy}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="MIN">Najnizszy koszt (MIN)</SelectItem>
                <SelectItem value="MAX">Najwyzszy koszt (MAX)</SelectItem>
                <SelectItem value="SUM">Suma kosztow (SUM)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <Label>Dostawa zagraniczna</Label>
              <p className="text-sm text-muted-foreground">
                Wlacz mozliwosc wysylki za granice
              </p>
            </div>
            <Switch
              checked={abroadEnabled}
              onCheckedChange={setAbroadEnabled}
            />
          </div>

          <div className="flex items-center justify-between">
            <div>
              <Label>Niestandardowy koszt dostawy</Label>
              <p className="text-sm text-muted-foreground">
                Pozwol kupujacym na negocjacje kosztu dostawy
              </p>
            </div>
            <Switch
              checked={customCostAllowed}
              onCheckedChange={setCustomCostAllowed}
            />
          </div>
        </div>

        <Button
          onClick={handleSave}
          disabled={updateSettings.isPending}
        >
          {updateSettings.isPending && (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          )}
          Zapisz ustawienia
        </Button>
      </CardContent>
    </Card>
  );
}

function ShippingRatesSection() {
  const { data, isLoading, isFetching } = useAllegroShippingRates();
  const [createOpen, setCreateOpen] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Truck className="h-5 w-5" />
            Cenniki wysylki
            {isFetching && (
              <Loader2 className="ml-2 h-4 w-4 animate-spin" />
            )}
          </CardTitle>
          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="mr-2 h-4 w-4" />
                Nowy cennik
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Nowy cennik wysylki</DialogTitle>
              </DialogHeader>
              <CreateShippingRateForm
                onSuccess={() => setCreateOpen(false)}
              />
            </DialogContent>
          </Dialog>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        ) : !data?.shippingRates?.length ? (
          <p className="py-8 text-center text-muted-foreground">
            Brak cennikow wysylki
          </p>
        ) : (
          <div className="space-y-2">
            {data.shippingRates.map((rate) => (
              <ShippingRateCard
                key={rate.id}
                rate={rate}
                expanded={expandedId === rate.id}
                onToggle={() =>
                  setExpandedId(
                    expandedId === rate.id ? null : rate.id
                  )
                }
              />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ShippingRateCard({
  rate,
  expanded,
  onToggle,
}: {
  rate: AllegroShippingRateSet;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <div className="rounded-lg border">
      <button
        className="flex w-full items-center justify-between p-4 text-left hover:bg-muted/50"
        onClick={onToggle}
      >
        <div>
          <p className="font-medium">{rate.name}</p>
          <p className="text-sm text-muted-foreground">
            {rate.rates?.length ?? 0} stawek &middot; ID: {rate.id}
          </p>
        </div>
        {expanded ? (
          <ChevronUp className="h-4 w-4 text-muted-foreground" />
        ) : (
          <ChevronDown className="h-4 w-4 text-muted-foreground" />
        )}
      </button>
      {expanded && rate.rates?.length > 0 && (
        <div className="border-t px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Metoda dostawy (ID)</TableHead>
                <TableHead className="w-28">Pierwsza szt.</TableHead>
                <TableHead className="w-28">Kolejna szt.</TableHead>
                <TableHead className="w-24">Max szt.</TableHead>
                <TableHead className="w-32">Czas wysylki</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rate.rates.map((entry, i) => (
                <TableRow key={i}>
                  <TableCell className="text-sm">
                    {entry.deliveryMethod.id}
                  </TableCell>
                  <TableCell>
                    {entry.firstItemRate.amount}{" "}
                    {entry.firstItemRate.currency}
                  </TableCell>
                  <TableCell>
                    {entry.nextItemRate.amount}{" "}
                    {entry.nextItemRate.currency}
                  </TableCell>
                  <TableCell>
                    {entry.maxQuantityPerPackage ?? "---"}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {entry.shippingTime
                      ? `${entry.shippingTime.from} - ${entry.shippingTime.to}`
                      : "---"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

function CreateShippingRateForm({
  onSuccess,
}: {
  onSuccess: () => void;
}) {
  const [name, setName] = useState("");
  const [deliveryMethodId, setDeliveryMethodId] = useState("");
  const [firstItemRate, setFirstItemRate] = useState("");
  const [nextItemRate, setNextItemRate] = useState("");

  const createRate = useCreateAllegroShippingRate();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) {
      toast.error("Nazwa cennika jest wymagana");
      return;
    }
    if (!deliveryMethodId.trim()) {
      toast.error("ID metody dostawy jest wymagane");
      return;
    }

    const data: AllegroCreateShippingRateRequest = {
      name: name.trim(),
      rates: [
        {
          deliveryMethod: { id: deliveryMethodId.trim() },
          firstItemRate: {
            amount: firstItemRate || "0",
            currency: "PLN",
          },
          nextItemRate: {
            amount: nextItemRate || "0",
            currency: "PLN",
          },
        },
      ],
    };

    createRate.mutate(data, {
      onSuccess: () => {
        toast.success("Cennik utworzony");
        setName("");
        setDeliveryMethodId("");
        setFirstItemRate("");
        setNextItemRate("");
        onSuccess();
      },
      onError: () =>
        toast.error("Nie udalo sie utworzyc cennika wysylki"),
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="rate-name">Nazwa cennika</Label>
        <Input
          id="rate-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="np. Cennik standardowy"
          required
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="delivery-method-id">ID metody dostawy</Label>
        <Input
          id="delivery-method-id"
          value={deliveryMethodId}
          onChange={(e) => setDeliveryMethodId(e.target.value)}
          placeholder="np. d8d7dbcb-6a04-4ddf-862f-..."
          required
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="first-item-rate">
            Pierwsza sztuka (PLN)
          </Label>
          <Input
            id="first-item-rate"
            type="number"
            step="0.01"
            min="0"
            value={firstItemRate}
            onChange={(e) => setFirstItemRate(e.target.value)}
            placeholder="9.99"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="next-item-rate">
            Kolejna sztuka (PLN)
          </Label>
          <Input
            id="next-item-rate"
            type="number"
            step="0.01"
            min="0"
            value={nextItemRate}
            onChange={(e) => setNextItemRate(e.target.value)}
            placeholder="5.99"
          />
        </div>
      </div>

      <Button
        type="submit"
        className="w-full"
        disabled={createRate.isPending}
      >
        {createRate.isPending && (
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        )}
        Utworz cennik
      </Button>
    </form>
  );
}

function DeliveryMethodsSection() {
  const { data, isLoading } = useAllegroDeliveryMethods();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Metody dostawy</CardTitle>
        <CardDescription>
          Dostepne metody dostawy na Allegro
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : !data?.deliveryMethods?.length ? (
          <p className="py-4 text-center text-muted-foreground">
            Brak metod dostawy
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nazwa</TableHead>
                <TableHead className="w-32">ID</TableHead>
                <TableHead className="w-40">Platnosc</TableHead>
                <TableHead className="w-32">Darmowa dostawa</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.deliveryMethods.map(
                (method: AllegroDeliveryMethodItem) => (
                  <TableRow key={method.id}>
                    <TableCell className="font-medium">
                      {method.name}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground font-mono">
                      {method.id.substring(0, 8)}...
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {method.paymentPolicy}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {method.shippingRatesConstraints
                        ?.allowedForFreeShipping ? (
                        <Badge variant="default">Tak</Badge>
                      ) : (
                        <Badge variant="secondary">Nie</Badge>
                      )}
                    </TableCell>
                  </TableRow>
                )
              )}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
