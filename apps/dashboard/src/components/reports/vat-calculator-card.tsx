"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
import { Calculator, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useVATRates, useCalculateVAT } from "@/hooks/use-vat-oss";
import { getErrorMessage } from "@/lib/api-client";

const RATE_TYPES = [
  { value: "standard", labelKey: "rateStandard" },
  { value: "reduced_1", labelKey: "rateReduced1" },
  { value: "reduced_2", labelKey: "rateReduced2" },
  { value: "super_reduced", labelKey: "rateSuperReduced" },
] as const;

function eur(value: number): string {
  return `${value.toLocaleString("pl-PL", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })} EUR`;
}

/** Standalone OSS VAT calculator backed by POST /v1/vat-oss/calculate. */
export function VatCalculatorCard() {
  const tf = useTranslations("feHooks");
  const { data: rates } = useVATRates();
  const calc = useCalculateVAT();

  const [amount, setAmount] = useState("");
  const [country, setCountry] = useState("");
  const [rateType, setRateType] = useState<string>("standard");

  const countryOptions = Object.keys(rates ?? {}).sort((a, b) =>
    a.localeCompare(b)
  );
  const result = calc.data;

  const handleCalculate = () => {
    const amt = Number(amount);
    if (!Number.isFinite(amt) || amt <= 0 || !country) {
      toast.error(tf("vatCalcInvalid"));
      return;
    }
    calc.mutate(
      { amount: amt, country, rate_type: rateType },
      { onError: (error) => toast.error(getErrorMessage(error)) }
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Calculator className="h-5 w-5" />
          {tf("vatCalcTitle")}
        </CardTitle>
        <CardDescription>{tf("vatCalcDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-3">
          <div className="space-y-1.5">
            <Label htmlFor="vat-calc-amount">{tf("vatCalcAmount")}</Label>
            <Input
              id="vat-calc-amount"
              type="number"
              min={0}
              step="0.01"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label>{tf("vatCalcCountry")}</Label>
            <Select value={country} onValueChange={setCountry}>
              <SelectTrigger>
                <SelectValue placeholder={tf("vatCalcSelectCountry")} />
              </SelectTrigger>
              <SelectContent>
                {countryOptions.map((code) => (
                  <SelectItem key={code} value={code}>
                    {code}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>{tf("vatCalcRateType")}</Label>
            <Select value={rateType} onValueChange={setRateType}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {RATE_TYPES.map((rt) => (
                  <SelectItem key={rt.value} value={rt.value}>
                    {tf(rt.labelKey)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <Button onClick={handleCalculate} disabled={calc.isPending}>
          {calc.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Calculator className="mr-2 h-4 w-4" />
          )}
          {tf("vatCalcCalculate")}
        </Button>

        {result && (
          <div className="grid gap-3 rounded-md border p-4 sm:grid-cols-4">
            <div>
              <p className="text-xs text-muted-foreground">{tf("vatCalcNet")}</p>
              <p className="font-medium">{eur(result.net_amount)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">{tf("vatCalcVat")}</p>
              <p className="font-medium">{eur(result.vat_amount)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">{tf("vatCalcGross")}</p>
              <p className="font-medium">{eur(result.gross_amount)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">{tf("vatCalcRate")}</p>
              <p className="font-medium">{result.vat_rate.toFixed(1)}%</p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
