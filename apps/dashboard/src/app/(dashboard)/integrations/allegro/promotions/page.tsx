"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  Plus,
  Trash2,
  Tag,
  Award,
  Info,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAllegroPromotions,
  useCreateAllegroPromotion,
  useDeleteAllegroPromotion,
  useAllegroPromoBadges,
} from "@/hooks/use-allegro";
import type {
  AllegroPromotion,
  AllegroCreatePromotionRequest,
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
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

const PAGE_SIZE = 20;

function statusBadgeVariant(status: string) {
  switch (status) {
    case "ACTIVE":
      return "default" as const;
    case "SUSPENDED":
      return "secondary" as const;
    case "FINISHED":
      return "destructive" as const;
    default:
      return "outline" as const;
  }
}

function statusLabel(status: string) {
  switch (status) {
    case "ACTIVE":
      return "Aktywna";
    case "SUSPENDED":
      return "Wstrzymana";
    case "FINISHED":
      return "Zakonczona";
    default:
      return status;
  }
}

function benefitTypeLabel(type?: string) {
  switch (type) {
    case "ORDER_FIXED_DISCOUNT":
      return "Rabat kwotowy";
    case "MULTI_PACK":
      return "Wielopak";
    case "FREE_SHIPPING":
      return "Darmowa dostawa";
    case "PERCENTAGE_DISCOUNT":
      return "Rabat procentowy";
    default:
      return type ?? "---";
  }
}

export default function AllegroPromotionsPage() {
  const [page, setPage] = useState(0);
  const [createOpen, setCreateOpen] = useState(false);

  const queryParams = {
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
  };

  const { data, isLoading, isFetching } = useAllegroPromotions(queryParams);
  const { data: badgesData, isLoading: badgesLoading } =
    useAllegroPromoBadges();

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/integrations/allegro">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div className="flex-1">
            <h1 className="text-2xl font-bold">Promocje Allegro</h1>
            <p className="text-muted-foreground">
              Zarzadzaj kampaniami promocyjnymi na Allegro
            </p>
          </div>
          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Nowa promocja
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Nowa promocja</DialogTitle>
              </DialogHeader>
              <CreatePromotionForm
                onSuccess={() => setCreateOpen(false)}
              />
            </DialogContent>
          </Dialog>
        </div>

        {/* Help section */}
        <div className="rounded-lg border bg-muted/50 p-4 flex gap-3">
          <Info className="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div className="space-y-1 text-sm">
            <p className="font-medium">Jak dzialaja promocje?</p>
            <ul className="list-disc list-inside space-y-0.5 text-muted-foreground">
              <li>Tworzac promocje wybierz typ rabatu: <strong>kwotowy</strong> (stala kwota znizki), <strong>procentowy</strong> (% od ceny), <strong>darmowa dostawa</strong> lub <strong>wielopak</strong> (rabat za kupno wielu sztuk).</li>
              <li>Po utworzeniu promocji musisz przypisac do niej oferty w panelu Allegro, aby kupujacy mogli z niej skorzystac.</li>
              <li>Statusy: <strong>Aktywna</strong> — dziala, <strong>Wstrzymana</strong> — tymczasowo wylaczona, <strong>Zakonczona</strong> — wygasla.</li>
              <li><strong>Pakiety promocyjne</strong> to platne uslugi Allegro (np. wyroznione w wynikach wyszukiwania) — ponizej znajdziesz dostepne pakiety z cenami.</li>
            </ul>
          </div>
        </div>

        {/* Promotions table */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Tag className="h-5 w-5" />
              Promocje
              {data && (
                <span className="text-sm font-normal text-muted-foreground">
                  ({data.count} lacznie)
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
            ) : !data?.promotions?.length ? (
              <p className="py-8 text-center text-muted-foreground">
                Brak promocji do wyswietlenia
              </p>
            ) : (
              <>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Nazwa</TableHead>
                      <TableHead className="w-40">Typ rabatu</TableHead>
                      <TableHead className="w-28">Status</TableHead>
                      <TableHead className="w-40">Data utworzenia</TableHead>
                      <TableHead className="w-24 text-right">Akcje</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.promotions.map((promo) => (
                      <PromotionRow key={promo.id} promotion={promo} />
                    ))}
                  </TableBody>
                </Table>

                {/* Pagination */}
                <div className="mt-4 flex items-center justify-between">
                  <p className="text-sm text-muted-foreground">
                    Strona {page + 1} z{" "}
                    {Math.max(1, Math.ceil(data.count / PAGE_SIZE))}
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
                      disabled={(page + 1) * PAGE_SIZE >= data.count}
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

        {/* Promotion badges */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Award className="h-5 w-5" />
              Pakiety promocyjne
            </CardTitle>
          </CardHeader>
          <CardContent>
            {badgesLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))}
              </div>
            ) : !badgesData?.packages?.length ? (
              <p className="py-4 text-center text-muted-foreground">
                Brak dostepnych pakietow promocyjnych
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Nazwa</TableHead>
                    <TableHead>Opis</TableHead>
                    <TableHead className="w-32 text-right">Cena</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {badgesData.packages.map((badge) => (
                    <TableRow key={badge.id}>
                      <TableCell className="font-medium">
                        {badge.name}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {badge.description ?? "---"}
                      </TableCell>
                      <TableCell className="text-right">
                        {badge.price.amount} {badge.price.currency}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}

function PromotionRow({ promotion }: { promotion: AllegroPromotion }) {
  const deletePromotion = useDeleteAllegroPromotion();

  const benefitType =
    promotion.benefits?.[0]?.specification?.type;

  const handleDelete = () => {
    deletePromotion.mutate(promotion.id, {
      onSuccess: () => toast.success("Promocja usunieta"),
      onError: () => toast.error("Nie udalo sie usunac promocji"),
    });
  };

  return (
    <TableRow>
      <TableCell>
        <p className="font-medium">{promotion.name}</p>
        <p className="text-xs text-muted-foreground">ID: {promotion.id}</p>
      </TableCell>
      <TableCell>
        <Badge variant="outline">{benefitTypeLabel(benefitType)}</Badge>
      </TableCell>
      <TableCell>
        <Badge variant={statusBadgeVariant(promotion.status)}>
          {statusLabel(promotion.status)}
        </Badge>
      </TableCell>
      <TableCell className="text-sm text-muted-foreground">
        {promotion.createdAt
          ? new Date(promotion.createdAt).toLocaleDateString("pl-PL")
          : "---"}
      </TableCell>
      <TableCell className="text-right">
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              disabled={deletePromotion.isPending}
            >
              {deletePromotion.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Trash2 className="h-4 w-4" />
              )}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Usunac promocje?</AlertDialogTitle>
              <AlertDialogDescription>
                Czy na pewno chcesz usunac promocje &quot;{promotion.name}
                &quot;? Tej operacji nie mozna cofnac.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Anuluj</AlertDialogCancel>
              <AlertDialogAction onClick={handleDelete}>
                Usun
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </TableCell>
    </TableRow>
  );
}

function CreatePromotionForm({
  onSuccess,
}: {
  onSuccess: () => void;
}) {
  const [name, setName] = useState("");
  const [benefitType, setBenefitType] = useState("ORDER_FIXED_DISCOUNT");
  const [discountAmount, setDiscountAmount] = useState("");
  const [discountCurrency, setDiscountCurrency] = useState("PLN");

  const createPromotion = useCreateAllegroPromotion();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) {
      toast.error("Nazwa promocji jest wymagana");
      return;
    }

    const data: AllegroCreatePromotionRequest = {
      name: name.trim(),
      benefits: [
        {
          specification: {
            type: benefitType,
            ...(benefitType !== "FREE_SHIPPING" && discountAmount
              ? {
                  value: {
                    amount: discountAmount,
                    currency: discountCurrency,
                  },
                }
              : {}),
          },
        },
      ],
    };

    createPromotion.mutate(data, {
      onSuccess: () => {
        toast.success("Promocja utworzona");
        setName("");
        setDiscountAmount("");
        onSuccess();
      },
      onError: () => toast.error("Nie udalo sie utworzyc promocji"),
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="promo-name">Nazwa promocji</Label>
        <Input
          id="promo-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="np. Letnia wyprzedaz"
          required
        />
      </div>

      <div className="space-y-2">
        <Label>Typ rabatu</Label>
        <Select value={benefitType} onValueChange={setBenefitType}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="ORDER_FIXED_DISCOUNT">
              Rabat kwotowy
            </SelectItem>
            <SelectItem value="PERCENTAGE_DISCOUNT">
              Rabat procentowy
            </SelectItem>
            <SelectItem value="FREE_SHIPPING">
              Darmowa dostawa
            </SelectItem>
            <SelectItem value="MULTI_PACK">
              Wielopak
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      {benefitType !== "FREE_SHIPPING" && (
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="discount-amount">Wartosc rabatu</Label>
            <Input
              id="discount-amount"
              type="number"
              step="0.01"
              min="0"
              value={discountAmount}
              onChange={(e) => setDiscountAmount(e.target.value)}
              placeholder="10.00"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="discount-currency">Waluta</Label>
            <Select
              value={discountCurrency}
              onValueChange={setDiscountCurrency}
            >
              <SelectTrigger id="discount-currency">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="PLN">PLN</SelectItem>
                <SelectItem value="EUR">EUR</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      )}

      <Button
        type="submit"
        className="w-full"
        disabled={createPromotion.isPending}
      >
        {createPromotion.isPending && (
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        )}
        Utworz promocje
      </Button>
    </form>
  );
}
