"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { API_URL, apiClient, getErrorMessage } from "@/lib/api-client";
import { usePublicConfig } from "@/hooks/use-public-config";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import type { PublicPlanInfo, CheckoutSessionResponse } from "@/types/api";

function formatPrice(amount: number, currency: string): string {
  return new Intl.NumberFormat("pl-PL", {
    style: "currency",
    currency: currency.toUpperCase(),
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  }).format(amount / 100);
}

function PricingContent() {
  const config = usePublicConfig();
  const router = useRouter();
  const [plans, setPlans] = useState<PublicPlanInfo[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [yearly, setYearly] = useState(false);
  const [loadingPlan, setLoadingPlan] = useState<string | null>(null);

  useEffect(() => {
    if (!config.billing_enabled) {
      setIsLoading(false);
      return;
    }

    fetch(`${API_URL}/v1/billing/plans`)
      .then((res) => res.json())
      .then((data: PublicPlanInfo[]) => {
        setPlans(data);
      })
      .catch(() => {
        toast.error("Nie udało się załadować planów");
      })
      .finally(() => setIsLoading(false));
  }, [config.billing_enabled]);

  // If billing not enabled, show the invite-based registration
  if (!config.billing_enabled) {
    // Redirect to invite registration form
    router.replace("/register/invite");
    return null;
  }

  const handleSelectPlan = async (planId: string) => {
    setLoadingPlan(planId);
    try {
      const res = await apiClient<CheckoutSessionResponse>("/v1/billing/checkout", {
        method: "POST",
        body: JSON.stringify({
          plan_id: planId,
          interval: yearly ? "year" : "month",
        }),
      });
      window.location.href = res.checkout_url;
    } catch (error) {
      toast.error(getErrorMessage(error));
      setLoadingPlan(null);
    }
  };

  if (isLoading) {
    return (
      <div className="w-full max-w-4xl mx-auto space-y-6">
        <div className="text-center space-y-2">
          <Skeleton className="h-8 w-64 mx-auto" />
          <Skeleton className="h-5 w-96 mx-auto" />
        </div>
        <div className="grid md:grid-cols-3 gap-6">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-80 rounded-lg" />
          ))}
        </div>
      </div>
    );
  }

  if (plans.length === 0) {
    return (
      <Card className="max-w-md mx-auto">
        <CardHeader className="text-center">
          <CardTitle>Brak dostępnych planów</CardTitle>
        </CardHeader>
        <CardContent className="text-center text-muted-foreground">
          <p>Plany cenowe nie są jeszcze skonfigurowane.</p>
        </CardContent>
        <CardFooter className="justify-center">
          <Link href="/login" className="text-sm text-primary underline-offset-4 hover:underline">
            Zaloguj się
          </Link>
        </CardFooter>
      </Card>
    );
  }

  return (
    <div className="w-full max-w-4xl mx-auto space-y-8">
      <div className="text-center space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Wybierz plan</h1>
        <p className="text-muted-foreground">
          Rozpocznij zarządzanie zamówieniami z OpenOMS
        </p>
      </div>

      <div className="flex items-center justify-center gap-3">
        <Label htmlFor="billing-interval" className={!yearly ? "font-semibold" : "text-muted-foreground"}>
          Miesięcznie
        </Label>
        <Switch
          id="billing-interval"
          checked={yearly}
          onCheckedChange={setYearly}
        />
        <Label htmlFor="billing-interval" className={yearly ? "font-semibold" : "text-muted-foreground"}>
          Rocznie
        </Label>
      </div>

      <div className="grid md:grid-cols-3 gap-6">
        {plans.map((plan) => {
          const price = yearly ? plan.yearly_amount : plan.monthly_amount;
          const perMonth = yearly
            ? Math.round(plan.yearly_amount / 12)
            : plan.monthly_amount;

          return (
            <Card key={plan.id} className="flex flex-col">
              <CardHeader className="text-center pb-2">
                <CardTitle className="text-xl">{plan.name}</CardTitle>
                <div className="mt-4">
                  <span className="text-3xl font-bold">
                    {formatPrice(perMonth, plan.currency)}
                  </span>
                  <span className="text-muted-foreground text-sm"> /mies.</span>
                </div>
                {yearly && (
                  <p className="text-xs text-muted-foreground mt-1">
                    {formatPrice(price, plan.currency)} rocznie
                  </p>
                )}
                {plan.trial_days > 0 && (
                  <p className="text-xs text-primary mt-2 font-medium">
                    {plan.trial_days} dni za darmo
                  </p>
                )}
              </CardHeader>
              <CardContent className="flex-1">
                <ul className="space-y-2 text-sm">
                  {plan.features.map((feature, i) => (
                    <li key={i} className="flex items-start gap-2">
                      <span className="text-primary mt-0.5 shrink-0">&#10003;</span>
                      <span>{feature}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
              <CardFooter>
                <Button
                  className="w-full"
                  onClick={() => handleSelectPlan(plan.id)}
                  disabled={loadingPlan !== null}
                >
                  {loadingPlan === plan.id ? "Przekierowanie..." : "Wybierz plan"}
                </Button>
              </CardFooter>
            </Card>
          );
        })}
      </div>

      <div className="text-center space-y-2 text-sm text-muted-foreground">
        <p>
          Masz już konto?{" "}
          <Link href="/login" className="text-primary underline-offset-4 hover:underline">
            Zaloguj się
          </Link>
        </p>
        <p>
          Masz token zaproszenia?{" "}
          <Link href="/register/invite" className="text-primary underline-offset-4 hover:underline">
            Zarejestruj się z zaproszeniem
          </Link>
        </p>
      </div>
    </div>
  );
}

export default function RegisterPage() {
  return (
    <Suspense>
      <PricingContent />
    </Suspense>
  );
}
