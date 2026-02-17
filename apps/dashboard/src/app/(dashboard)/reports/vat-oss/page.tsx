"use client";

import { useState, useCallback } from "react";
import {
  Download,
  Globe,
  Receipt,
  Loader2,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  useOSSReport,
  useVATRates,
  useDownloadOSSReportCSV,
} from "@/hooks/use-vat-oss";

const QUARTERS = [
  { value: "1", label: "Q1 (sty-mar)" },
  { value: "2", label: "Q2 (kwi-cze)" },
  { value: "3", label: "Q3 (lip-wrz)" },
  { value: "4", label: "Q4 (paz-gru)" },
];

const QUARTER_DEADLINES: Record<number, string> = {
  1: "30 kwietnia",
  2: "31 lipca",
  3: "31 pazdziernika",
  4: "31 stycznia (nastepnego roku)",
};

// Country flag emoji from country code
function countryFlag(code: string): string {
  const codePoints = code
    .toUpperCase()
    .split("")
    .map((char) => 127397 + char.charCodeAt(0));
  return String.fromCodePoint(...codePoints);
}

export default function VATOSSReportPage() {
  const now = new Date();
  const currentQuarter = Math.ceil((now.getMonth() + 1) / 3);
  const currentYear = now.getFullYear();

  const [quarter, setQuarter] = useState(currentQuarter);
  const [year, setYear] = useState(currentYear);
  const [showRatesTable, setShowRatesTable] = useState(false);

  const { data: report, isLoading: reportLoading } = useOSSReport(
    quarter,
    year
  );
  const { data: allRates, isLoading: ratesLoading } = useVATRates();
  const downloadCSV = useDownloadOSSReportCSV();

  const handleDownload = useCallback(() => {
    downloadCSV(quarter, year);
  }, [downloadCSV, quarter, year]);

  // Generate year options: 2021 to current year + 1
  const yearOptions = [];
  for (let y = currentYear + 1; y >= 2021; y--) {
    yearOptions.push(y);
  }

  const totalCountries = report?.by_country?.length ?? 0;
  const totalOrders = report?.by_country?.reduce((sum, c) => sum + c.order_count, 0) ?? 0;

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <Receipt className="h-6 w-6" />
              Raport VAT OSS
            </h1>
            <p className="text-muted-foreground mt-1">
              Kwartalne zestawienie sprzedazy transgranicznej w UE z obliczeniem
              VAT
            </p>
          </div>
          <Button
            variant="outline"
            onClick={handleDownload}
            disabled={!report || report.by_country.length === 0}
          >
            <Download className="h-4 w-4" />
            Pobierz CSV
          </Button>
        </div>

        {/* Quarter/Year selector */}
        <Card>
          <CardContent className="py-4">
            <div className="flex flex-wrap items-center gap-4">
              <div className="flex items-center gap-2">
                <label className="text-sm font-medium">Kwartal:</label>
                <Select
                  value={String(quarter)}
                  onValueChange={(v) => setQuarter(parseInt(v, 10))}
                >
                  <SelectTrigger className="w-[160px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {QUARTERS.map((q) => (
                      <SelectItem key={q.value} value={q.value}>
                        {q.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="flex items-center gap-2">
                <label className="text-sm font-medium">Rok:</label>
                <Select
                  value={String(year)}
                  onValueChange={(v) => setYear(parseInt(v, 10))}
                >
                  <SelectTrigger className="w-[100px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {yearOptions.map((y) => (
                      <SelectItem key={y} value={String(y)}>
                        {y}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <p className="text-xs text-muted-foreground ml-auto">
                Termin deklaracji: {QUARTER_DEADLINES[quarter]}
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Summary cards */}
        {reportLoading ? (
          <div className="grid gap-4 md:grid-cols-3">
            {[...Array(3)].map((_, i) => (
              <Skeleton key={i} className="h-28 w-full" />
            ))}
          </div>
        ) : report ? (
          <div className="grid gap-4 md:grid-cols-3">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Sprzedaz transgraniczna
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">
                  {report.total_sales.toLocaleString("pl-PL", {
                    minimumFractionDigits: 2,
                    maximumFractionDigits: 2,
                  })}{" "}
                  EUR
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {totalOrders} zamowien w Q{quarter} {year}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Nalezny VAT
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">
                  {report.total_vat.toLocaleString("pl-PL", {
                    minimumFractionDigits: 2,
                    maximumFractionDigits: 2,
                  })}{" "}
                  EUR
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  VAT do zadeklarowania w ramach OSS
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Kraje sprzedazy
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold flex items-center gap-2">
                  <Globe className="h-5 w-5 text-muted-foreground" />
                  {totalCountries}
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  krajow UE z zamowieniami
                </p>
              </CardContent>
            </Card>
          </div>
        ) : null}

        {/* Country breakdown table */}
        {reportLoading ? (
          <Skeleton className="h-64 w-full" />
        ) : report && report.by_country.length > 0 ? (
          <Card>
            <CardHeader>
              <CardTitle>Zestawienie wg krajow</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left py-2 pr-4 font-medium">Kraj</th>
                      <th className="text-right py-2 px-4 font-medium">
                        Zamowienia
                      </th>
                      <th className="text-right py-2 px-4 font-medium">
                        Sprzedaz (EUR)
                      </th>
                      <th className="text-right py-2 px-4 font-medium">
                        Stawka VAT
                      </th>
                      <th className="text-right py-2 pl-4 font-medium">
                        Kwota VAT (EUR)
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {report.by_country.map((cr) => (
                      <tr
                        key={cr.country}
                        className="border-b last:border-0"
                      >
                        <td className="py-2 pr-4 font-medium">
                          <span className="mr-2">{countryFlag(cr.country)}</span>
                          {cr.country_name}{" "}
                          <span className="text-muted-foreground">
                            ({cr.country})
                          </span>
                        </td>
                        <td className="text-right py-2 px-4">
                          {cr.order_count}
                        </td>
                        <td className="text-right py-2 px-4">
                          {cr.total_sales.toLocaleString("pl-PL", {
                            minimumFractionDigits: 2,
                          })}
                        </td>
                        <td className="text-right py-2 px-4">
                          {cr.vat_rate.toFixed(1)}%
                        </td>
                        <td className="text-right py-2 pl-4 font-medium">
                          {cr.vat_amount.toLocaleString("pl-PL", {
                            minimumFractionDigits: 2,
                          })}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr className="border-t font-semibold">
                      <td className="py-2 pr-4">RAZEM</td>
                      <td className="text-right py-2 px-4">{totalOrders}</td>
                      <td className="text-right py-2 px-4">
                        {report.total_sales.toLocaleString("pl-PL", {
                          minimumFractionDigits: 2,
                        })}
                      </td>
                      <td className="text-right py-2 px-4" />
                      <td className="text-right py-2 pl-4">
                        {report.total_vat.toLocaleString("pl-PL", {
                          minimumFractionDigits: 2,
                        })}
                      </td>
                    </tr>
                  </tfoot>
                </table>
              </div>
            </CardContent>
          </Card>
        ) : report ? (
          <Card>
            <CardContent className="py-8">
              <p className="text-center text-muted-foreground">
                Brak sprzedazy transgranicznej w Q{quarter} {year}
              </p>
            </CardContent>
          </Card>
        ) : null}

        {/* EU VAT rates reference table */}
        <Card>
          <CardHeader
            className="cursor-pointer"
            onClick={() => setShowRatesTable((prev) => !prev)}
          >
            <CardTitle className="flex items-center justify-between">
              <span>Stawki VAT w krajach UE</span>
              {showRatesTable ? (
                <ChevronUp className="h-4 w-4 text-muted-foreground" />
              ) : (
                <ChevronDown className="h-4 w-4 text-muted-foreground" />
              )}
            </CardTitle>
          </CardHeader>
          {showRatesTable && (
            <CardContent>
              {ratesLoading ? (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Ladowanie stawek...
                </div>
              ) : allRates ? (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b">
                        <th className="text-left py-2 pr-4 font-medium">
                          Kraj
                        </th>
                        <th className="text-right py-2 px-4 font-medium">
                          Standardowa
                        </th>
                        <th className="text-right py-2 px-4 font-medium">
                          Obnizona 1
                        </th>
                        <th className="text-right py-2 px-4 font-medium">
                          Obnizona 2
                        </th>
                        <th className="text-right py-2 pl-4 font-medium">
                          Super obnizona
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {Object.entries(allRates)
                        .sort(([a], [b]) => a.localeCompare(b))
                        .map(([code, rates]) => (
                          <tr
                            key={code}
                            className="border-b last:border-0"
                          >
                            <td className="py-2 pr-4 font-medium">
                              <span className="mr-2">
                                {countryFlag(code)}
                              </span>
                              {code}
                            </td>
                            <td className="text-right py-2 px-4">
                              {rates.standard.toFixed(1)}%
                            </td>
                            <td className="text-right py-2 px-4">
                              {rates.reduced_1
                                ? `${rates.reduced_1.toFixed(1)}%`
                                : "—"}
                            </td>
                            <td className="text-right py-2 px-4">
                              {rates.reduced_2
                                ? `${rates.reduced_2.toFixed(1)}%`
                                : "—"}
                            </td>
                            <td className="text-right py-2 pl-4">
                              {rates.super_reduced != null && rates.super_reduced > 0
                                ? `${rates.super_reduced.toFixed(1)}%`
                                : "—"}
                            </td>
                          </tr>
                        ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <p className="text-muted-foreground">
                  Nie udalo sie zaladowac stawek VAT
                </p>
              )}
            </CardContent>
          )}
        </Card>

        {/* Disclaimer */}
        <p className="text-xs text-muted-foreground text-center">
          Raport jest informacyjny i nie zastepuje oficjalnej deklaracji OSS.
          Stawki VAT sa aktualizowane okresowo — zweryfikuj biezace stawki
          na stronie Komisji Europejskiej przed zlozeniem deklaracji.
        </p>
      </div>
    </AdminGuard>
  );
}
