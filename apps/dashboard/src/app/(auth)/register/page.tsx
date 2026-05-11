"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Check, Sparkles, ArrowRight, Loader2, Package } from "lucide-react";
import { API_URL, apiClient, getErrorMessage } from "@/lib/api-client";
import { usePublicConfig } from "@/hooks/use-public-config";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { PublicPlanInfo, CheckoutSessionResponse } from "@/types/api";

function formatPrice(amount: number, currency: string): string {
  return new Intl.NumberFormat("pl-PL", {
    style: "currency",
    currency: currency.toUpperCase(),
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
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
    if (config.isLoading) {
      return;
    }
    if (!config.billing_enabled) {
      setIsLoading(false);
      return;
    }

    fetch(`${API_URL}/v1/billing/plans`, { credentials: "include" })
      .then((res) => res.json())
      .then((data: PublicPlanInfo[]) => {
        setPlans(data);
      })
      .catch(() => {
        toast.error("Nie udalo sie zaladowac planow");
      })
      .finally(() => setIsLoading(false));
  }, [config.billing_enabled, config.isLoading]);

  useEffect(() => {
    if (!config.isLoading && !config.billing_enabled) {
      router.replace("/register/invite");
    }
  }, [config.billing_enabled, config.isLoading, router]);

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
      // eslint-disable-next-line react-hooks/immutability
      window.location.href = res.checkout_url;
    } catch (error) {
      toast.error(getErrorMessage(error));
      setLoadingPlan(null);
    }
  };

  if (config.isLoading || isLoading) {
    return (
      <div className="max-w-5xl mx-auto space-y-10 py-12">
        <div className="text-center space-y-3">
          <Skeleton className="h-10 w-72 mx-auto" />
          <Skeleton className="h-5 w-80 mx-auto" />
        </div>
        <Skeleton className="h-10 w-64 mx-auto rounded-full" />
        <div className="grid md:grid-cols-3 gap-6 px-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-[460px] rounded-2xl" />
          ))}
        </div>
      </div>
    );
  }

  if (!config.billing_enabled) {
    return null;
  }

  if (plans.length === 0) {
    return (
      <div className="max-w-md mx-auto text-center py-20 space-y-4">
        <Package className="size-12 mx-auto text-muted-foreground/50" />
        <h2 className="text-xl font-semibold">Brak dostepnych planow</h2>
        <p className="text-muted-foreground text-sm">
          Plany cenowe nie sa jeszcze skonfigurowane.
        </p>
        <Link href="/login" className="text-sm text-primary underline-offset-4 hover:underline">
          Zaloguj sie
        </Link>
      </div>
    );
  }

  // Middle plan is the "featured" one
  const featuredIndex = plans.length >= 3 ? 1 : 0;
  const trialDays = plans[0]?.trial_days ?? 0;
  const yearlySavingsPercent = plans[0]
    ? Math.round(100 - (plans[0].yearly_amount / (plans[0].monthly_amount * 12)) * 100)
    : 0;

  return (
    <div className="max-w-5xl mx-auto py-8 md:py-12 px-4">
      {/* Header */}
      <div className="text-center space-y-3 mb-10">
        {trialDays > 0 && (
          <div className="inline-flex items-center gap-2 text-xs font-medium tracking-widest uppercase text-muted-foreground mb-2">
            <div className="size-1.5 rounded-full bg-success animate-pulse" />
            {trialDays} dni za darmo
          </div>
        )}
        <h1 className="text-3xl md:text-4xl font-bold tracking-tight">
          Wybierz plan dla swojego biznesu
        </h1>
        {trialDays > 0 && (
          <p className="text-muted-foreground max-w-lg mx-auto">
            Wszystkie plany z pelnym dostepem przez {trialDays} dni. Bez karty na start.
            Zrezygnuj kiedy chcesz.
          </p>
        )}
      </div>

      {/* Interval toggle */}
      <div className="flex items-center justify-center gap-3 mb-10">
        <Label
          htmlFor="billing-interval"
          className={cn(
            "text-sm cursor-pointer transition-colors",
            !yearly ? "text-foreground font-medium" : "text-muted-foreground"
          )}
        >
          Miesiecznie
        </Label>
        <Switch
          id="billing-interval"
          checked={yearly}
          onCheckedChange={setYearly}
        />
        <Label
          htmlFor="billing-interval"
          className={cn(
            "text-sm cursor-pointer transition-colors",
            yearly ? "text-foreground font-medium" : "text-muted-foreground"
          )}
        >
          Rocznie
        </Label>
        {yearly && yearlySavingsPercent > 0 && (
          <Badge variant="success" className="text-[11px] ml-1">
            -{yearlySavingsPercent}%
          </Badge>
        )}
      </div>

      {/* Plan cards */}
      <div className="grid md:grid-cols-3 gap-5 items-start">
        {plans.map((plan, index) => {
          const isFeatured = index === featuredIndex;
          const perMonth = yearly
            ? Math.round(plan.yearly_amount / 12)
            : plan.monthly_amount;
          const totalYearly = plan.yearly_amount;
          const isCurrentLoading = loadingPlan === plan.id;

          return (
            <div
              key={plan.id}
              className={cn(
                "relative flex flex-col rounded-2xl border bg-card text-card-foreground transition-all duration-200",
                isFeatured
                  ? "border-primary/20 shadow-[0_0_0_1px_var(--primary),0_8px_40px_-12px_oklch(0.3_0.05_250/0.25)] md:scale-[1.03] z-10"
                  : "border-border hover:border-border/80 shadow-sm hover:shadow-md",
              )}
            >
              {/* Featured badge */}
              {isFeatured && (
                <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                  <Badge className="bg-primary text-primary-foreground shadow-md text-[11px] px-3 py-0.5">
                    <Sparkles className="size-3 mr-1" />
                    Najpopularniejszy
                  </Badge>
                </div>
              )}

              {/* Card header */}
              <div className={cn("p-6 pb-4", isFeatured && "pt-8")}>
                <h3 className="text-lg font-semibold">{plan.name}</h3>

                {/* Price */}
                <div className="mt-4 flex items-baseline gap-1">
                  <span className="text-4xl font-bold tracking-tight tabular-nums">
                    {formatPrice(perMonth, plan.currency)}
                  </span>
                  <span className="text-muted-foreground text-sm">/mies.</span>
                </div>

                {yearly && (
                  <p className="text-xs text-muted-foreground mt-1">
                    {formatPrice(totalYearly, plan.currency)} rocznie
                  </p>
                )}

                {plan.trial_days > 0 && (
                  <p className="text-xs text-success font-medium mt-2">
                    {plan.trial_days} dni za darmo na start
                  </p>
                )}
              </div>

              {/* Divider */}
              <div className="mx-6 border-t" />

              {/* Features */}
              <div className="p-6 pt-4 flex-1">
                <ul className="space-y-2.5">
                  {plan.features.map((feature, i) => (
                    <li key={i} className="flex items-start gap-2.5 text-sm">
                      <Check className="size-4 shrink-0 mt-0.5 text-success" />
                      <span>{feature}</span>
                    </li>
                  ))}
                </ul>
              </div>

              {/* CTA */}
              <div className="p-6 pt-2">
                <Button
                  className={cn(
                    "w-full h-11 font-medium",
                    isFeatured
                      ? ""
                      : "bg-secondary text-secondary-foreground hover:bg-secondary/80",
                  )}
                  onClick={() => handleSelectPlan(plan.id)}
                  disabled={loadingPlan !== null}
                >
                  {isCurrentLoading ? (
                    <>
                      <Loader2 className="size-4 mr-2 animate-spin" />
                      Przekierowanie...
                    </>
                  ) : (
                    <>
                      Zacznij za darmo
                      <ArrowRight className="size-4 ml-2" />
                    </>
                  )}
                </Button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Footer links */}
      <div className="mt-10 text-center space-y-2 text-sm text-muted-foreground">
        <p>
          Masz juz konto?{" "}
          <Link href="/login" className="text-foreground font-medium underline-offset-4 hover:underline">
            Zaloguj sie
          </Link>
        </p>
        <p>
          Masz token zaproszenia?{" "}
          <Link href="/register/invite" className="text-foreground font-medium underline-offset-4 hover:underline">
            Zarejestruj sie z zaproszeniem
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
