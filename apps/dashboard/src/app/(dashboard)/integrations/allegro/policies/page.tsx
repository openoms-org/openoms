"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  Plus,
  Pencil,
  Trash2,
  ChevronDown,
  ChevronRight,
  Shield,
  Ruler,
  RotateCcw,
  Info,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAllegroReturnPolicies,
  useCreateAllegroReturnPolicy,
  useUpdateAllegroReturnPolicy,
  useAllegroWarranties,
  useCreateAllegroWarranty,
  useUpdateAllegroWarranty,
  useAllegroSizeTables,
  useCreateAllegroSizeTable,
  useUpdateAllegroSizeTable,
  useDeleteAllegroSizeTable,
} from "@/hooks/use-allegro";
import type {
  AllegroReturnPolicy,
  AllegroCreateReturnPolicyRequest,
  AllegroImpliedWarranty,
  AllegroCreateWarrantyRequest,
  AllegroSizeTable,
  AllegroCreateSizeTableRequest,
} from "@/hooks/use-allegro";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

export default function AllegroPoliciesPage() {
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
            <h1 className="text-2xl font-bold">Polityki sprzedazy</h1>
            <p className="text-muted-foreground">
              Zarzadzaj politykami zwrotow, rekojmia i tabelami rozmiarow
            </p>
          </div>
        </div>

        {/* Help section */}
        <div className="rounded-lg border bg-muted/50 p-4 flex gap-3">
          <Info className="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div className="space-y-1 text-sm">
            <p className="font-medium">Polityki sprzedazy — co tu ustawisz?</p>
            <ul className="list-disc list-inside space-y-0.5 text-muted-foreground">
              <li><strong>Polityki zwrotow</strong> — okresl czas na zwrot (np. 14 lub 30 dni), kto pokrywa koszty zwrotu (sprzedawca/kupujacy) i podaj adres do zwrotow. Kazda oferta musi miec przypisana polityke.</li>
              <li><strong>Rekojmia</strong> — ustaw okres ochrony rekojmi dla osob fizycznych i firm. Standardowo: 2 lata dla konsumentow, 1 rok dla firm.</li>
              <li><strong>Tabele rozmiarow</strong> — stworz tabele z rozmiarami (np. obuwie, odziez) i przypisz je do ofert. Pomagaja kupujacym wybrac odpowiedni rozmiar i zmniejszaja ilosc zwrotow.</li>
              <li>Utworzone polityki i tabele przypisujesz do ofert na stronie &quot;Oferty&quot;.</li>
            </ul>
          </div>
        </div>

        <Tabs defaultValue="return-policies">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="return-policies" className="flex items-center gap-2">
              <RotateCcw className="h-4 w-4" />
              Polityki zwrotow
            </TabsTrigger>
            <TabsTrigger value="warranties" className="flex items-center gap-2">
              <Shield className="h-4 w-4" />
              Rekojmia
            </TabsTrigger>
            <TabsTrigger value="size-tables" className="flex items-center gap-2">
              <Ruler className="h-4 w-4" />
              Tabele rozmiarow
            </TabsTrigger>
          </TabsList>

          <TabsContent value="return-policies">
            <ReturnPoliciesTab />
          </TabsContent>
          <TabsContent value="warranties">
            <WarrantiesTab />
          </TabsContent>
          <TabsContent value="size-tables">
            <SizeTablesTab />
          </TabsContent>
        </Tabs>
      </div>
    </AdminGuard>
  );
}

// ===================== Return Policies Tab =====================

function ReturnPoliciesTab() {
  const { data, isLoading } = useAllegroReturnPolicies();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingPolicy, setEditingPolicy] = useState<AllegroReturnPolicy | null>(null);

  const handleEdit = (policy: AllegroReturnPolicy) => {
    setEditingPolicy(policy);
    setDialogOpen(true);
  };

  const handleCreate = () => {
    setEditingPolicy(null);
    setDialogOpen(true);
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <RotateCcw className="h-5 w-5" />
          Polityki zwrotow
        </CardTitle>
        <Button size="sm" onClick={handleCreate}>
          <Plus className="mr-1 h-4 w-4" />
          Nowa polityka
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : !data?.returnPolicies?.length ? (
          <p className="py-8 text-center text-muted-foreground">
            Brak polityk zwrotow
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nazwa</TableHead>
                <TableHead className="w-32">Czas na zwrot</TableHead>
                <TableHead className="w-32">Koszty zwrotu</TableHead>
                <TableHead>Opis</TableHead>
                <TableHead className="w-24 text-right">Akcje</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.returnPolicies.map((policy) => (
                <TableRow key={policy.id}>
                  <TableCell className="font-medium">{policy.name}</TableCell>
                  <TableCell>
                    {policy.availability
                      ? `${policy.availability.range} dni`
                      : "---"}
                  </TableCell>
                  <TableCell>
                    {policy.returnCost?.coveredBy ? (
                      <Badge
                        variant={
                          policy.returnCost.coveredBy === "SELLER"
                            ? "default"
                            : "secondary"
                        }
                      >
                        {policy.returnCost.coveredBy === "SELLER"
                          ? "Sprzedawca"
                          : "Kupujacy"}
                      </Badge>
                    ) : (
                      "---"
                    )}
                  </TableCell>
                  <TableCell className="max-w-xs truncate text-sm text-muted-foreground">
                    {policy.description || "---"}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleEdit(policy)}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>

      <ReturnPolicyDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        policy={editingPolicy}
      />
    </Card>
  );
}

function ReturnPolicyDialog({
  open,
  onOpenChange,
  policy,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  policy: AllegroReturnPolicy | null;
}) {
  const createMutation = useCreateAllegroReturnPolicy();
  const updateMutation = useUpdateAllegroReturnPolicy();
  const isEditing = !!policy;

  const [name, setName] = useState("");
  const [rangeVal, setRangeVal] = useState("14");
  const [availType, setAvailType] = useState("FULL");
  const [coveredBy, setCoveredBy] = useState("SELLER");
  const [description, setDescription] = useState("");
  const [addrName, setAddrName] = useState("");
  const [addrStreet, setAddrStreet] = useState("");
  const [addrCity, setAddrCity] = useState("");
  const [addrZip, setAddrZip] = useState("");
  const [addrCountry, setAddrCountry] = useState("PL");

  // Reset form when dialog opens
  const handleOpenChange = (val: boolean) => {
    if (val && policy) {
      setName(policy.name);
      // withdrawalPeriod is ISO 8601 "P14D" -> extract days
      const wp = policy.withdrawalPeriod ?? "P14D";
      const daysMatch = wp.match(/P(\d+)D/);
      setRangeVal(daysMatch ? daysMatch[1] : "14");
      setAvailType(policy.availability?.range ?? "FULL");
      setCoveredBy(policy.returnCost?.coveredBy ?? "SELLER");
      setDescription(policy.description ?? "");
      setAddrName(policy.address?.name ?? "");
      setAddrStreet(policy.address?.street ?? "");
      setAddrCity(policy.address?.city ?? "");
      setAddrZip(policy.address?.postCode ?? "");
      setAddrCountry(policy.address?.countryCode ?? "PL");
    } else if (val) {
      setName("");
      setRangeVal("14");
      setAvailType("FULL");
      setCoveredBy("SELLER");
      setDescription("");
      setAddrName("");
      setAddrStreet("");
      setAddrCity("");
      setAddrZip("");
      setAddrCountry("PL");
    }
    onOpenChange(val);
  };

  const handleSubmit = () => {
    if (!name.trim()) {
      toast.error("Nazwa jest wymagana");
      return;
    }

    const data: AllegroCreateReturnPolicyRequest = {
      name: name.trim(),
      availability: { range: availType },
      withdrawalPeriod: `P${rangeVal}D`,
      returnCost: { coveredBy },
      options: {
        cashOnDeliveryNotAllowed: false,
        freeAccessoriesReturnRequired: false,
        refundLoweredByReceivedDiscount: false,
        businessReturnAllowed: false,
        collectBySellerOnly: false,
      },
      description: description.trim() || undefined,
    };

    if (addrStreet.trim()) {
      data.address = {
        name: addrName.trim(),
        street: addrStreet.trim(),
        city: addrCity.trim(),
        postCode: addrZip.trim(),
        countryCode: addrCountry,
      };
    }

    if (isEditing) {
      updateMutation.mutate(
        { policyId: policy.id, data },
        {
          onSuccess: () => {
            toast.success("Polityka zwrotow zaktualizowana");
            onOpenChange(false);
          },
          onError: () => toast.error("Nie udalo sie zaktualizowac polityki"),
        }
      );
    } else {
      createMutation.mutate(data, {
        onSuccess: () => {
          toast.success("Polityka zwrotow utworzona");
          onOpenChange(false);
        },
        onError: () => toast.error("Nie udalo sie utworzyc polityki"),
      });
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? "Edytuj polityke zwrotow" : "Nowa polityka zwrotow"}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Nazwa</Label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="np. Standardowa polityka zwrotow"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Czas na zwrot (dni)</Label>
              <Input
                type="number"
                min="1"
                value={rangeVal}
                onChange={(e) => setRangeVal(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Koszty zwrotu</Label>
              <Select value={coveredBy} onValueChange={setCoveredBy}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="SELLER">Sprzedawca</SelectItem>
                  <SelectItem value="BUYER">Kupujacy</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="space-y-2">
            <Label>Opis</Label>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Dodatkowe informacje o polityce zwrotow..."
              rows={3}
            />
          </div>
          <div className="space-y-2">
            <Label className="text-sm font-medium">Adres do zwrotow</Label>
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Nazwa"
                value={addrName}
                onChange={(e) => setAddrName(e.target.value)}
              />
              <Input
                placeholder="Kod kraju (np. PL)"
                value={addrCountry}
                onChange={(e) => setAddrCountry(e.target.value)}
              />
            </div>
            <Input
              placeholder="Ulica"
              value={addrStreet}
              onChange={(e) => setAddrStreet(e.target.value)}
            />
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Miasto"
                value={addrCity}
                onChange={(e) => setAddrCity(e.target.value)}
              />
              <Input
                placeholder="Kod pocztowy"
                value={addrZip}
                onChange={(e) => setAddrZip(e.target.value)}
              />
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Anuluj
          </Button>
          <Button onClick={handleSubmit} disabled={isPending}>
            {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isEditing ? "Zapisz" : "Utworz"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ===================== Warranties Tab =====================

const WARRANTY_DURATION_OPTIONS = [
  { value: "P1Y", label: "1 rok" },
  { value: "P2Y", label: "2 lata" },
  { value: "P3Y", label: "3 lata" },
  { value: "P5Y", label: "5 lat" },
];

function durationLabel(duration?: string): string {
  const found = WARRANTY_DURATION_OPTIONS.find((o) => o.value === duration);
  return found ? found.label : duration ?? "---";
}

function WarrantiesTab() {
  const { data, isLoading } = useAllegroWarranties();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingWarranty, setEditingWarranty] =
    useState<AllegroImpliedWarranty | null>(null);

  const handleEdit = (warranty: AllegroImpliedWarranty) => {
    setEditingWarranty(warranty);
    setDialogOpen(true);
  };

  const handleCreate = () => {
    setEditingWarranty(null);
    setDialogOpen(true);
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <Shield className="h-5 w-5" />
          Rekojmia
        </CardTitle>
        <Button size="sm" onClick={handleCreate}>
          <Plus className="mr-1 h-4 w-4" />
          Nowa rekojmia
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : !data?.impliedWarranties?.length ? (
          <p className="py-8 text-center text-muted-foreground">
            Brak rekojmi
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nazwa</TableHead>
                <TableHead className="w-48">
                  Okres gwarancji (osoby fizyczne)
                </TableHead>
                <TableHead className="w-48">
                  Okres gwarancji (firmy)
                </TableHead>
                <TableHead className="w-24 text-right">Akcje</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.impliedWarranties.map((warranty) => (
                <TableRow key={warranty.id}>
                  <TableCell className="font-medium">{warranty.name}</TableCell>
                  <TableCell>
                    {durationLabel(warranty.individual?.period)}
                  </TableCell>
                  <TableCell>
                    {durationLabel(warranty.corporate?.period)}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleEdit(warranty)}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>

      <WarrantyDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        warranty={editingWarranty}
      />
    </Card>
  );
}

function WarrantyDialog({
  open,
  onOpenChange,
  warranty,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  warranty: AllegroImpliedWarranty | null;
}) {
  const createMutation = useCreateAllegroWarranty();
  const updateMutation = useUpdateAllegroWarranty();
  const isEditing = !!warranty;

  const [name, setName] = useState("");
  const [individualDuration, setIndividualDuration] = useState("P2Y");
  const [individualImplied, setIndividualImplied] = useState(true);
  const [corporateDuration, setCorporateDuration] = useState("P1Y");
  const [corporateImplied, setCorporateImplied] = useState(true);

  const handleOpenChange = (val: boolean) => {
    if (val && warranty) {
      setName(warranty.name);
      setIndividualDuration(warranty.individual?.period ?? "P2Y");
      setIndividualImplied(warranty.individual?.type === "IMPLIED_WARRANTY" || !warranty.individual?.type);
      setCorporateDuration(warranty.corporate?.period ?? "P2Y");
      setCorporateImplied(warranty.corporate?.type === "IMPLIED_WARRANTY" || !warranty.corporate?.type);
    } else if (val) {
      setName("");
      setIndividualDuration("P2Y");
      setIndividualImplied(true);
      setCorporateDuration("P1Y");
      setCorporateImplied(true);
    }
    onOpenChange(val);
  };

  const handleSubmit = () => {
    if (!name.trim()) {
      toast.error("Nazwa jest wymagana");
      return;
    }

    const data: AllegroCreateWarrantyRequest = {
      name: name.trim(),
      individual: { period: individualDuration, type: individualImplied ? "IMPLIED_WARRANTY" : "WITHOUT_WARRANTY" },
      corporate: { period: corporateDuration, type: corporateImplied ? "IMPLIED_WARRANTY" : "WITHOUT_WARRANTY" },
    };

    if (isEditing) {
      updateMutation.mutate(
        { warrantyId: warranty.id, data },
        {
          onSuccess: () => {
            toast.success("Rekojmia zaktualizowana");
            onOpenChange(false);
          },
          onError: () => toast.error("Nie udalo sie zaktualizowac rekojmi"),
        }
      );
    } else {
      createMutation.mutate(data, {
        onSuccess: () => {
          toast.success("Rekojmia utworzona");
          onOpenChange(false);
        },
        onError: () => toast.error("Nie udalo sie utworzyc rekojmi"),
      });
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? "Edytuj rekojmie" : "Nowa rekojmia"}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Nazwa</Label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="np. Standardowa rekojmia"
            />
          </div>
          <div className="space-y-2">
            <Label>Okres gwarancji (osoby fizyczne)</Label>
            <Select
              value={individualDuration}
              onValueChange={setIndividualDuration}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {WARRANTY_DURATION_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Okres gwarancji (firmy)</Label>
            <Select
              value={corporateDuration}
              onValueChange={setCorporateDuration}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {WARRANTY_DURATION_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Anuluj
          </Button>
          <Button onClick={handleSubmit} disabled={isPending}>
            {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isEditing ? "Zapisz" : "Utworz"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ===================== Size Tables Tab =====================

const SIZE_TABLE_TYPES = [
  { value: "MALE", label: "Mezczyzni" },
  { value: "FEMALE", label: "Kobiety" },
  { value: "KIDS", label: "Dzieci" },
  { value: "UNISEX", label: "Unisex" },
];

function sizeTypeLabel(type: string): string {
  const found = SIZE_TABLE_TYPES.find((t) => t.value === type);
  return found ? found.label : type;
}

function SizeTablesTab() {
  const { data, isLoading } = useAllegroSizeTables();
  const deleteMutation = useDeleteAllegroSizeTable();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingTable, setEditingTable] = useState<AllegroSizeTable | null>(
    null
  );
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const handleEdit = (table: AllegroSizeTable) => {
    setEditingTable(table);
    setDialogOpen(true);
  };

  const handleCreate = () => {
    setEditingTable(null);
    setDialogOpen(true);
  };

  const handleDelete = (table: AllegroSizeTable) => {
    if (!confirm(`Czy na pewno chcesz usunac tabele "${table.name}"?`)) return;
    deleteMutation.mutate(table.id, {
      onSuccess: () => toast.success("Tabela rozmiarow usunieta"),
      onError: () => toast.error("Nie udalo sie usunac tabeli"),
    });
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <Ruler className="h-5 w-5" />
          Tabele rozmiarow
        </CardTitle>
        <Button size="sm" onClick={handleCreate}>
          <Plus className="mr-1 h-4 w-4" />
          Nowa tabela
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : !data?.sizeTables?.length ? (
          <p className="py-8 text-center text-muted-foreground">
            Brak tabel rozmiarow
          </p>
        ) : (
          <div className="space-y-2">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-10" />
                  <TableHead>Nazwa</TableHead>
                  <TableHead className="w-28">Typ</TableHead>
                  <TableHead className="w-28">Rozmiary</TableHead>
                  <TableHead className="w-28 text-right">Akcje</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.sizeTables.map((table) => (
                  <>
                    <TableRow key={table.id}>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0"
                          onClick={() =>
                            setExpandedId(
                              expandedId === table.id ? null : table.id
                            )
                          }
                        >
                          {expandedId === table.id ? (
                            <ChevronDown className="h-4 w-4" />
                          ) : (
                            <ChevronRight className="h-4 w-4" />
                          )}
                        </Button>
                      </TableCell>
                      <TableCell className="font-medium">
                        {table.name}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">
                          {sizeTypeLabel(table.type)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {table.values?.length ?? 0} wierszy
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleEdit(table)}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(table)}
                            disabled={deleteMutation.isPending}
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                    {expandedId === table.id && (
                      <TableRow key={`${table.id}-expanded`}>
                        <TableCell colSpan={5} className="bg-muted/50 p-4">
                          <SizeTablePreview table={table} />
                        </TableCell>
                      </TableRow>
                    )}
                  </>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>

      <SizeTableDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        table={editingTable}
      />
    </Card>
  );
}

function SizeTablePreview({ table }: { table: AllegroSizeTable }) {
  if (!table.headers?.length) {
    return (
      <p className="text-sm text-muted-foreground">Brak danych w tabeli</p>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-sm border">
        <thead>
          <tr className="bg-muted">
            {table.headers.map((h, i) => (
              <th
                key={i}
                className="px-3 py-2 text-left font-medium border-r last:border-r-0"
              >
                {h.name}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {table.values?.map((row, ri) => (
            <tr key={ri} className="border-t">
              {row.map((cell, ci) => (
                <td
                  key={ci}
                  className="px-3 py-1.5 border-r last:border-r-0"
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function SizeTableDialog({
  open,
  onOpenChange,
  table,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  table: AllegroSizeTable | null;
}) {
  const createMutation = useCreateAllegroSizeTable();
  const updateMutation = useUpdateAllegroSizeTable();
  const isEditing = !!table;

  const [name, setName] = useState("");
  const [type, setType] = useState("UNISEX");
  const [headers, setHeaders] = useState<string[]>(["Rozmiar"]);
  const [values, setValues] = useState<string[][]>([[""]]);

  const handleOpenChange = (val: boolean) => {
    if (val && table) {
      setName(table.name);
      setType(table.type);
      setHeaders(table.headers?.map((h) => h.name) ?? ["Rozmiar"]);
      setValues(
        table.values?.length
          ? table.values.map((row) => [...row])
          : [[""]]
      );
    } else if (val) {
      setName("");
      setType("UNISEX");
      setHeaders(["Rozmiar"]);
      setValues([[""]]);
    }
    onOpenChange(val);
  };

  const addColumn = () => {
    setHeaders([...headers, ""]);
    setValues(values.map((row) => [...row, ""]));
  };

  const removeColumn = (index: number) => {
    if (headers.length <= 1) return;
    setHeaders(headers.filter((_, i) => i !== index));
    setValues(values.map((row) => row.filter((_, i) => i !== index)));
  };

  const addRow = () => {
    setValues([...values, headers.map(() => "")]);
  };

  const removeRow = (index: number) => {
    if (values.length <= 1) return;
    setValues(values.filter((_, i) => i !== index));
  };

  const updateHeader = (index: number, value: string) => {
    const next = [...headers];
    next[index] = value;
    setHeaders(next);
  };

  const updateCell = (rowIndex: number, colIndex: number, value: string) => {
    const next = values.map((row) => [...row]);
    next[rowIndex][colIndex] = value;
    setValues(next);
  };

  const handleSubmit = () => {
    if (!name.trim()) {
      toast.error("Nazwa jest wymagana");
      return;
    }
    if (headers.some((h) => !h.trim())) {
      toast.error("Wszystkie naglowki musza byc wypelnione");
      return;
    }

    const data: AllegroCreateSizeTableRequest = {
      name: name.trim(),
      type,
      headers: headers.map((h) => ({ name: h.trim() })),
      values,
    };

    if (isEditing) {
      updateMutation.mutate(
        { tableId: table.id, data },
        {
          onSuccess: () => {
            toast.success("Tabela rozmiarow zaktualizowana");
            onOpenChange(false);
          },
          onError: () => toast.error("Nie udalo sie zaktualizowac tabeli"),
        }
      );
    } else {
      createMutation.mutate(data, {
        onSuccess: () => {
          toast.success("Tabela rozmiarow utworzona");
          onOpenChange(false);
        },
        onError: () => toast.error("Nie udalo sie utworzyc tabeli"),
      });
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? "Edytuj tabele rozmiarow" : "Nowa tabela rozmiarow"}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Nazwa</Label>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="np. Buty meskie - EU"
              />
            </div>
            <div className="space-y-2">
              <Label>Typ</Label>
              <Select value={type} onValueChange={setType}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SIZE_TABLE_TYPES.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Dynamic table builder */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Naglowki i wartosci</Label>
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={addColumn}
                >
                  <Plus className="mr-1 h-3 w-3" />
                  Dodaj kolumne
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={addRow}
                >
                  <Plus className="mr-1 h-3 w-3" />
                  Dodaj wiersz
                </Button>
              </div>
            </div>

            <div className="overflow-x-auto border rounded-md">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="bg-muted">
                    {headers.map((h, i) => (
                      <th key={i} className="px-2 py-1.5 border-r last:border-r-0">
                        <div className="flex items-center gap-1">
                          <Input
                            value={h}
                            onChange={(e) => updateHeader(i, e.target.value)}
                            className="h-7 text-xs"
                            placeholder="Naglowek"
                          />
                          {headers.length > 1 && (
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              className="h-7 w-7 p-0 shrink-0"
                              onClick={() => removeColumn(i)}
                            >
                              <Trash2 className="h-3 w-3 text-destructive" />
                            </Button>
                          )}
                        </div>
                      </th>
                    ))}
                    <th className="w-10 px-1" />
                  </tr>
                </thead>
                <tbody>
                  {values.map((row, ri) => (
                    <tr key={ri} className="border-t">
                      {row.map((cell, ci) => (
                        <td key={ci} className="px-2 py-1 border-r last:border-r-0">
                          <Input
                            value={cell}
                            onChange={(e) =>
                              updateCell(ri, ci, e.target.value)
                            }
                            className="h-7 text-xs"
                          />
                        </td>
                      ))}
                      <td className="px-1">
                        {values.length > 1 && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="h-7 w-7 p-0"
                            onClick={() => removeRow(ri)}
                          >
                            <Trash2 className="h-3 w-3 text-destructive" />
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Anuluj
          </Button>
          <Button onClick={handleSubmit} disabled={isPending}>
            {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isEditing ? "Zapisz" : "Utworz"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
