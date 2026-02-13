"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Calculator,
  CreditCard,
  Loader2,
  Search,
  Info,
} from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useAllegroBilling, useAllegroFees } from "@/hooks/use-allegro";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const PAGE_SIZE = 20;

export default function AllegroFinancePage() {
  return (
    <AdminGuard>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/integrations/allegro">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Finanse Allegro</h1>
            <p className="text-muted-foreground">
              Rozliczenia i kalkulator prowizji
            </p>
          </div>
        </div>

        {/* Help section */}
        <div className="rounded-lg border bg-muted/50 p-4 flex gap-3">
          <Info className="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div className="space-y-1 text-sm">
            <p className="font-medium">Informacje o finansach</p>
            <ul className="list-disc list-inside space-y-0.5 text-muted-foreground">
              <li><strong>Rozliczenia</strong> — historia wpisow rozliczeniowych z Allegro (prowizje, wplaty, zwroty). Mozesz filtrowac po grupie typu, np. INCOME (przychod) lub REFUND (zwrot).</li>
              <li><strong>Kalkulator prowizji</strong> — wpisz ID oferty Allegro, aby sprawdzic ile wyniesie prowizja i jakie oplaty zostana naliczone. Przydatne przed ustaleniem ceny koncowej.</li>
              <li>Dane finansowe sa pobierane w czasie rzeczywistym z API Allegro.</li>
            </ul>
          </div>
        </div>

        <Tabs defaultValue="billing">
          <TabsList>
            <TabsTrigger value="billing">
              <CreditCard className="mr-2 h-4 w-4" />
              Rozliczenia
            </TabsTrigger>
            <TabsTrigger value="calculator">
              <Calculator className="mr-2 h-4 w-4" />
              Kalkulator prowizji
            </TabsTrigger>
          </TabsList>

          <TabsContent value="billing">
            <BillingSection />
          </TabsContent>

          <TabsContent value="calculator">
            <FeeCalculatorSection />
          </TabsContent>
        </Tabs>
      </div>
    </AdminGuard>
  );
}

function BillingSection() {
  const [page, setPage] = useState(0);
  const [typeGroup, setTypeGroup] = useState<string>("");

  const { data, isLoading, isFetching } = useAllegroBilling({
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
    type_group: typeGroup || undefined,
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <CreditCard className="h-5 w-5" />
          Rozliczenia
          {data && (
            <span className="text-sm font-normal text-muted-foreground">
              ({data.count} wpisow)
            </span>
          )}
          {isFetching && <Loader2 className="ml-2 h-4 w-4 animate-spin" />}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {/* Type group filter */}
        <div className="mb-4 flex items-center gap-3">
          <Input
            placeholder="Filtruj po grupie typu (np. INCOME, REFUND)..."
            value={typeGroup}
            onChange={(e) => {
              setTypeGroup(e.target.value);
              setPage(0);
            }}
            className="max-w-xs"
          />
        </div>

        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : !data?.billingEntries?.length ? (
          <p className="py-8 text-center text-muted-foreground">
            Brak wpisow rozliczeniowych
          </p>
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Data</TableHead>
                  <TableHead>Typ</TableHead>
                  <TableHead>Grupa</TableHead>
                  <TableHead className="text-right">Kwota</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.billingEntries.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell className="text-sm">
                      {entry.occurredAt
                        ? new Date(entry.occurredAt).toLocaleDateString("pl-PL", {
                            year: "numeric",
                            month: "short",
                            day: "numeric",
                            hour: "2-digit",
                            minute: "2-digit",
                          })
                        : "---"}
                    </TableCell>
                    <TableCell>
                      <span className="text-sm">{entry.type.name}</span>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="text-xs">
                        {entry.type.group}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right font-mono text-sm">
                      <span
                        className={
                          parseFloat(entry.amount.amount) < 0
                            ? "text-red-600"
                            : "text-green-600"
                        }
                      >
                        {entry.amount.amount} {entry.amount.currency}
                      </span>
                    </TableCell>
                  </TableRow>
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
  );
}

function FeeCalculatorSection() {
  const [offerIdInput, setOfferIdInput] = useState("");
  const [activeOfferId, setActiveOfferId] = useState<string | null>(null);

  const { data, isLoading, isError } = useAllegroFees(activeOfferId);

  const handleCalculate = () => {
    const trimmed = offerIdInput.trim();
    if (trimmed) {
      setActiveOfferId(trimmed);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Calculator className="h-5 w-5" />
          Kalkulator prowizji
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">
          Wpisz ID oferty Allegro, aby obliczyc prowizje i oplaty.
        </p>

        <div className="flex flex-col gap-3 sm:flex-row">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="ID oferty Allegro..."
              value={offerIdInput}
              onChange={(e) => setOfferIdInput(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleCalculate()}
              className="pl-10"
            />
          </div>
          <Button
            onClick={handleCalculate}
            disabled={!offerIdInput.trim() || isLoading}
          >
            {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Oblicz prowizje
          </Button>
        </div>

        {isError && activeOfferId && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3">
            <p className="text-sm text-destructive">
              Nie udalo sie obliczyc prowizji dla oferty {activeOfferId}.
              Sprawdz, czy ID jest poprawne.
            </p>
          </div>
        )}

        {data && (
          <div className="space-y-4">
            {/* Commissions */}
            {data.commissions?.length > 0 && (
              <div>
                <h3 className="mb-2 font-semibold text-sm">Prowizje</h3>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Typ</TableHead>
                      <TableHead className="text-right">Stawka</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.commissions.map((comm, idx) => (
                      <TableRow key={idx}>
                        <TableCell className="text-sm">{comm.type}</TableCell>
                        <TableCell className="text-right font-mono text-sm">
                          {comm.rate.amount} {comm.rate.currency}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            {/* Quotes */}
            {data.quotes?.length > 0 && (
              <div>
                <h3 className="mb-2 font-semibold text-sm">Oplaty</h3>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Nazwa</TableHead>
                      <TableHead>Typ</TableHead>
                      <TableHead className="text-right">Oplata</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.quotes.map((quote, idx) => (
                      <TableRow key={idx}>
                        <TableCell className="text-sm">{quote.name}</TableCell>
                        <TableCell>
                          <Badge variant="outline" className="text-xs">
                            {quote.type}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm">
                          {quote.fee.amount} {quote.fee.currency}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            {!data.commissions?.length && !data.quotes?.length && (
              <p className="py-4 text-center text-muted-foreground">
                Brak danych o prowizjach dla tej oferty
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
