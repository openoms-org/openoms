"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { ChevronDown, ChevronUp, Loader2, Truck, Zap, Tag } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { SHIPMENT_PROVIDER_LABELS } from "@/lib/constants";
import { useShippingRates } from "@/hooks/use-shipping-rates";
import type { ShippingRate } from "@/types/api";

const rateFormSchema = z.object({
  from_postal_code: z.string().min(1, "Kod pocztowy nadawcy jest wymagany"),
  from_country: z.string(),
  to_postal_code: z.string().min(1, "Kod pocztowy odbiorcy jest wymagany"),
  to_country: z.string(),
  weight: z.number().positive("Waga musi być większa od 0"),
  width: z.number().min(0),
  height: z.number().min(0),
  length: z.number().min(0),
  cod: z.number().min(0),
});

type RateFormValues = z.infer<typeof rateFormSchema>;

interface RateShoppingProps {
  defaultToPostalCode?: string;
  defaultFromPostalCode?: string;
  defaultWeight?: number;
  defaultWidth?: number;
  defaultHeight?: number;
  defaultLength?: number;
  onSelectRate?: (rate: ShippingRate) => void;
}

export function RateShopping({
  defaultToPostalCode,
  defaultFromPostalCode,
  defaultWeight,
  defaultWidth,
  defaultHeight,
  defaultLength,
  onSelectRate,
}: RateShoppingProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [rates, setRates] = useState<ShippingRate[]>([]);
  const [hasSearched, setHasSearched] = useState(false);
  const [useCod, setUseCod] = useState(false);

  const [error, setError] = useState<string | null>(null);
  const getRates = useShippingRates();

  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<RateFormValues>({
    resolver: zodResolver(rateFormSchema),
    defaultValues: {
      from_postal_code: defaultFromPostalCode ?? "",
      from_country: "PL",
      to_postal_code: defaultToPostalCode ?? "",
      to_country: "PL",
      weight: defaultWeight ?? 0,
      width: defaultWidth ?? 0,
      height: defaultHeight ?? 0,
      length: defaultLength ?? 0,
      cod: 0,
    },
  });

  const onSubmit = (data: RateFormValues) => {
    if (!useCod) {
      data.cod = 0;
    }
    setError(null);
    getRates.mutate(
      {
        from_postal_code: data.from_postal_code,
        from_country: data.from_country,
        to_postal_code: data.to_postal_code,
        to_country: data.to_country,
        weight: data.weight,
        width: data.width,
        height: data.height,
        length: data.length,
        cod: data.cod,
      },
      {
        onSuccess: (response) => {
          setRates(response.rates);
          setHasSearched(true);
        },
        onError: (err) => {
          setError(err instanceof Error ? err.message : "Nie udało się pobrać stawek");
          setHasSearched(true);
        },
      }
    );
  };

  // Find cheapest and fastest
  const cheapestPrice =
    rates.length > 0 ? Math.min(...rates.map((r) => r.price)) : null;
  const fastestDays =
    rates.length > 0 ? Math.min(...rates.map((r) => r.estimated_days)) : null;

  return (
    <Card>
      <CardHeader
        className="cursor-pointer select-none"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <CardTitle className="flex items-center justify-between">
          <span className="flex items-center gap-2">
            <Truck className="h-5 w-5" />
            Porownaj ceny kurierow
          </span>
          {isExpanded ? (
            <ChevronUp className="h-5 w-5 text-muted-foreground" />
          ) : (
            <ChevronDown className="h-5 w-5 text-muted-foreground" />
          )}
        </CardTitle>
      </CardHeader>

      {isExpanded && (
        <CardContent className="space-y-6">
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="from_postal_code">
                  Kod pocztowy nadawcy
                </Label>
                <Input
                  id="from_postal_code"
                  placeholder="00-000"
                  {...register("from_postal_code")}
                />
                {errors.from_postal_code && (
                  <p className="text-sm text-destructive">
                    {errors.from_postal_code.message}
                  </p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="to_postal_code">
                  Kod pocztowy odbiorcy
                </Label>
                <Input
                  id="to_postal_code"
                  placeholder="00-000"
                  {...register("to_postal_code")}
                />
                {errors.to_postal_code && (
                  <p className="text-sm text-destructive">
                    {errors.to_postal_code.message}
                  </p>
                )}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
              <div className="space-y-2">
                <Label htmlFor="weight">Waga (kg)</Label>
                <Input
                  id="weight"
                  type="number"
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  {...register("weight", { valueAsNumber: true })}
                />
                {errors.weight && (
                  <p className="text-sm text-destructive">
                    {errors.weight.message}
                  </p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="width">Szer. (cm)</Label>
                <Input
                  id="width"
                  type="number"
                  step="0.1"
                  min="0"
                  placeholder="0"
                  {...register("width", { valueAsNumber: true })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="height">Wys. (cm)</Label>
                <Input
                  id="height"
                  type="number"
                  step="0.1"
                  min="0"
                  placeholder="0"
                  {...register("height", { valueAsNumber: true })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="length">Dl. (cm)</Label>
                <Input
                  id="length"
                  type="number"
                  step="0.1"
                  min="0"
                  placeholder="0"
                  {...register("length", { valueAsNumber: true })}
                />
              </div>
            </div>

            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <Checkbox
                  id="cod_checkbox"
                  checked={useCod}
                  onCheckedChange={(checked) => {
                    setUseCod(!!checked);
                    if (!checked) {
                      setValue("cod", 0);
                    }
                  }}
                />
                <Label htmlFor="cod_checkbox" className="cursor-pointer">
                  Pobranie (COD)
                </Label>
              </div>
              {useCod && (
                <div className="flex items-center gap-2">
                  <Input
                    type="number"
                    step="0.01"
                    min="0"
                    placeholder="Kwota pobrania"
                    className="w-40"
                    {...register("cod", { valueAsNumber: true })}
                  />
                  <span className="text-sm text-muted-foreground">PLN</span>
                </div>
              )}
            </div>

            <Button type="submit" disabled={getRates.isPending}>
              {getRates.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Porownywanie...
                </>
              ) : (
                "Porownaj ceny"
              )}
            </Button>
          </form>

          {error && (
            <div className="rounded-md border border-destructive bg-destructive/10 p-4 text-sm text-destructive">
              {error}
            </div>
          )}

          {hasSearched && !error && rates.length === 0 && (
            <div className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
              Brak dostepnych stawek dla podanych parametrow. Upewnij sie, ze masz aktywne integracje z kurierami.
            </div>
          )}

          {rates.length > 0 && (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Kurier</TableHead>
                    <TableHead>Usluga</TableHead>
                    <TableHead className="text-right">Cena</TableHead>
                    <TableHead className="text-center">
                      Szacowany czas dostawy
                    </TableHead>
                    <TableHead className="text-center">Typ</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rates.map((rate, index) => {
                    const isCheapest = rate.price === cheapestPrice;
                    const isFastest = rate.estimated_days === fastestDays;

                    return (
                      <TableRow
                        key={`${rate.carrier_code}-${rate.service_name}-${index}`}
                        className={
                          isCheapest
                            ? "bg-green-50 dark:bg-green-950/20"
                            : undefined
                        }
                      >
                        <TableCell className="font-medium">
                          <div className="flex items-center gap-2">
                            {SHIPMENT_PROVIDER_LABELS[rate.carrier_code] ??
                              rate.carrier_name}
                            {isCheapest && (
                              <Badge className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                                <Tag className="mr-1 h-3 w-3" />
                                Najtansza
                              </Badge>
                            )}
                            {isFastest && !isCheapest && (
                              <Badge className="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                                <Zap className="mr-1 h-3 w-3" />
                                Najszybsza
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>{rate.service_name}</TableCell>
                        <TableCell className="text-right font-semibold">
                          {rate.price.toFixed(2)} {rate.currency}
                        </TableCell>
                        <TableCell className="text-center">
                          {rate.estimated_days}{" "}
                          {rate.estimated_days === 1
                            ? "dzien roboczy"
                            : "dni roboczych"}
                        </TableCell>
                        <TableCell className="text-center">
                          {rate.pickup_point ? (
                            <Badge variant="outline">Punkt odbioru</Badge>
                          ) : (
                            <Badge variant="secondary">Kurier</Badge>
                          )}
                        </TableCell>
                        <TableCell>
                          {onSelectRate && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => onSelectRate(rate)}
                            >
                              Wybierz
                            </Button>
                          )}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      )}
    </Card>
  );
}
