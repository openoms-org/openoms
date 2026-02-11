"use client";

import { useState } from "react";
import { Coins, Trash2, Plus, Download } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useExchangeRates,
  useDeleteExchangeRate,
  useCreateExchangeRate,
  useFetchNBPRates,
} from "@/hooks/use-exchange-rates";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const CURRENCIES = ["PLN", "EUR", "USD", "GBP", "CZK", "CHF", "SEK", "NOK", "DKK", "HUF"];

export default function CurrenciesPage() {
  const { data, isLoading, isError, refetch } = useExchangeRates({ limit: 100 });
  const deleteRate = useDeleteExchangeRate();
  const createRate = useCreateExchangeRate();
  const fetchNBP = useFetchNBPRates();

  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [baseCurrency, setBaseCurrency] = useState("PLN");
  const [targetCurrency, setTargetCurrency] = useState("EUR");
  const [rate, setRate] = useState("");

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const rates = data?.items ?? [];

  const handleDelete = () => {
    if (!deleteId) return;
    deleteRate.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Kurs zostal usuniety");
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const handleCreate = () => {
    const rateNum = parseFloat(rate);
    if (!rateNum || rateNum <= 0) {
      toast.error("Kurs musi byc liczba dodatnia");
      return;
    }
    createRate.mutate(
      {
        base_currency: baseCurrency,
        target_currency: targetCurrency,
        rate: rateNum,
        source: "manual",
      },
      {
        onSuccess: () => {
          toast.success("Kurs zostal dodany");
          setShowCreate(false);
          setRate("");
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      }
    );
  };

  const handleFetchNBP = () => {
    fetchNBP.mutate(undefined, {
      onSuccess: (data) => {
        toast.success(`Pobrano ${data.fetched} kursow z NBP`);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  return (
    <AdminGuard>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Waluty i kursy wymiany</h1>
          <p className="text-muted-foreground">
            Zarzadzaj kursami walut i przeliczaj kwoty
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleFetchNBP} disabled={fetchNBP.isPending}>
            <Download className="h-4 w-4 mr-2" />
            {fetchNBP.isPending ? "Pobieranie..." : "Pobierz kursy NBP"}
          </Button>
          <Dialog open={showCreate} onOpenChange={setShowCreate}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Nowy kurs
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Nowy kurs wymiany</DialogTitle>
                <DialogDescription>
                  Dodaj recznie kurs wymiany walut
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>Waluta bazowa</Label>
                  <Select value={baseCurrency} onValueChange={setBaseCurrency}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {CURRENCIES.map((c) => (
                        <SelectItem key={c} value={c}>
                          {c}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>Waluta docelowa</Label>
                  <Select value={targetCurrency} onValueChange={setTargetCurrency}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {CURRENCIES.filter((c) => c !== baseCurrency).map((c) => (
                        <SelectItem key={c} value={c}>
                          {c}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>Kurs</Label>
                  <Input
                    type="number"
                    step="0.000001"
                    value={rate}
                    onChange={(e) => setRate(e.target.value)}
                    placeholder="np. 0.2326"
                  />
                  <p className="text-xs text-muted-foreground">
                    1 {baseCurrency} = ? {targetCurrency}
                  </p>
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setShowCreate(false)}>
                  Anuluj
                </Button>
                <Button
                  onClick={handleCreate}
                  disabled={!rate || createRate.isPending}
                >
                  {createRate.isPending ? "Dodawanie..." : "Dodaj"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4">
          <p className="text-sm text-destructive">
            Wystapil blad podczas ladowania danych. Sprobuj odswiezyc strone.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Sprobuj ponownie
          </Button>
        </div>
      )}

      {rates.length === 0 ? (
        <EmptyState
          icon={Coins}
          title="Brak kursow walut"
          description="Dodaj pierwszy kurs recznie lub pobierz kursy z NBP."
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Waluta bazowa</TableHead>
                <TableHead>Waluta docelowa</TableHead>
                <TableHead>Kurs</TableHead>
                <TableHead>Zrodlo</TableHead>
                <TableHead>Pobrano</TableHead>
                <TableHead className="w-[80px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rates.map((exchangeRate) => (
                <TableRow key={exchangeRate.id}>
                  <TableCell className="font-medium">
                    {exchangeRate.base_currency}
                  </TableCell>
                  <TableCell className="font-medium">
                    {exchangeRate.target_currency}
                  </TableCell>
                  <TableCell className="font-mono">
                    {exchangeRate.rate.toFixed(6)}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant="outline"
                      className={
                        exchangeRate.source === "nbp"
                          ? "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
                          : ""
                      }
                    >
                      {exchangeRate.source === "nbp" ? "NBP" : "Reczny"}
                    </Badge>
                  </TableCell>
                  <TableCell>{formatDate(exchangeRate.fetched_at)}</TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={() => setDeleteId(exchangeRate.id)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Usun kurs"
        description="Czy na pewno chcesz usunac ten kurs wymiany?"
        confirmLabel="Usun"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteRate.isPending}
      />
    </AdminGuard>
  );
}
