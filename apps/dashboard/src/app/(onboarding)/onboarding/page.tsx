"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { CheckCircle2, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import {
  useOnboardingStatus,
  useUpdateOnboardingStep,
  useCompleteOnboarding,
  useDismissOnboarding,
} from "@/hooks/use-onboarding-wizard";
import { apiClient } from "@/lib/api-client";
import { useTranslations } from "next-intl";

// ---------------------------------------------------------------------------
// Step labels for the stepper
// ---------------------------------------------------------------------------
const STEPS = [
  { number: 1, label: "Dane firmy" },
  { number: 2, label: "Magazyn" },
  { number: 3, label: "Integracja" },
  { number: 4, label: "Zespół" },
];

// ---------------------------------------------------------------------------
// Stepper header
// ---------------------------------------------------------------------------
function Stepper({ currentStep, completedSteps }: { currentStep: number; completedSteps: number[] }) {
  return (
    <div className="flex items-center gap-2 mb-8">
      {STEPS.map((step, idx) => {
        const isCompleted = completedSteps.includes(step.number);
        const isActive = step.number === currentStep;

        return (
          <div key={step.number} className="flex items-center flex-1">
            <div className="flex flex-col items-center gap-1">
              <div
                className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium transition-colors ${
                  isCompleted
                    ? "bg-primary text-primary-foreground"
                    : isActive
                    ? "border-2 border-primary text-primary"
                    : "border-2 border-muted-foreground/30 text-muted-foreground"
                }`}
              >
                {isCompleted ? <CheckCircle2 className="h-4 w-4" /> : step.number}
              </div>
              <span
                className={`text-xs whitespace-nowrap ${
                  isActive ? "text-foreground font-medium" : "text-muted-foreground"
                }`}
              >
                {step.label}
              </span>
            </div>
            {idx < STEPS.length - 1 && (
              <div
                className={`flex-1 h-0.5 mx-2 mt-[-1rem] ${
                  completedSteps.includes(step.number) ? "bg-primary" : "bg-muted"
                }`}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Step 1: Company details
// ---------------------------------------------------------------------------
function Step1Company({ onNext }: { onNext: () => void }) {
  const t = useTranslations("onboarding");
  const updateStep = useUpdateOnboardingStep();
  const [form, setForm] = useState({
    company_name: "",
    nip: "",
    address: "",
    city: "",
    post_code: "",
    phone: "",
    email: "",
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const validateNip = (raw: string): boolean => {
    const digits = raw.replace(/[-\s]/g, "");
    if (!/^\d{10}$/.test(digits)) return false;
    const weights = [6, 5, 7, 2, 3, 4, 5, 6, 7];
    const sum = weights.reduce((acc, w, i) => acc + w * Number(digits[i]), 0);
    const checksum = sum % 11;
    if (checksum === 10) return false;
    return checksum === Number(digits[9]);
  };

  const validate = () => {
    const e: Record<string, string> = {};
    if (!form.company_name.trim()) e.company_name = "Nazwa firmy jest wymagana";
    if (!form.nip.trim()) e.nip = "NIP jest wymagany";
    else if (!validateNip(form.nip)) e.nip = "Podaj poprawny numer NIP";
    if (form.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email))
      e.email = "Podaj poprawny adres email";
    if (form.post_code && !/^\d{2}-\d{3}$/.test(form.post_code))
      e.post_code = t("kodPocztowyMusiMiecFormatXxxxx");
    if (form.phone && !/^\+?[\d\s\-()]{7,20}$/.test(form.phone))
      e.phone = "Podaj poprawny numer telefonu";
    return e;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const errs = validate();
    if (Object.keys(errs).length > 0) {
      setErrors(errs);
      return;
    }
    try {
      await apiClient("/v1/settings/company", {
        method: "PUT",
        body: JSON.stringify({
          company_name: form.company_name,
          nip: form.nip,
          address: form.address,
          city: form.city,
          post_code: form.post_code,
          phone: form.phone,
          email: form.email,
        }),
      });
      await updateStep.mutateAsync({ step: 1, action: "completed" });
      onNext();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("error"));
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="company_name">
            Nazwa firmy <span className="text-destructive">*</span>
          </Label>
          <Input
            id="company_name"
            value={form.company_name}
            onChange={(e) => setForm({ ...form, company_name: e.target.value })}
            aria-invalid={!!errors.company_name}
          />
          {errors.company_name && (
            <p className="text-xs text-destructive">{errors.company_name}</p>
          )}
        </div>
        <div className="space-y-2">
          <Label htmlFor="nip">
            NIP <span className="text-destructive">*</span>
          </Label>
          <Input
            id="nip"
            value={form.nip}
            onChange={(e) => setForm({ ...form, nip: e.target.value })}
            placeholder="1234567890"
            aria-invalid={!!errors.nip}
          />
          {errors.nip && <p className="text-xs text-destructive">{errors.nip}</p>}
        </div>
        <div className="space-y-2 sm:col-span-2">
          <Label htmlFor="address">Ulica i numer</Label>
          <Input
            id="address"
            value={form.address}
            onChange={(e) => setForm({ ...form, address: e.target.value })}
            placeholder={t("form.streetPlaceholder")}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="city">Miasto</Label>
          <Input
            id="city"
            value={form.city}
            onChange={(e) => setForm({ ...form, city: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="post_code">Kod pocztowy</Label>
          <Input
            id="post_code"
            value={form.post_code}
            onChange={(e) => setForm({ ...form, post_code: e.target.value })}
            placeholder="00-000"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="phone">Telefon</Label>
          <Input
            id="phone"
            value={form.phone}
            onChange={(e) => setForm({ ...form, phone: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input
            id="email"
            type="email"
            value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
          />
        </div>
      </div>
      <div className="flex justify-end pt-2">
        <Button type="submit" disabled={updateStep.isPending}>
          {updateStep.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Dalej
        </Button>
      </div>
    </form>
  );
}

// ---------------------------------------------------------------------------
// Step 2: Warehouse
// ---------------------------------------------------------------------------
function Step2Warehouse({ onNext, onSkip }: { onNext: () => void; onSkip: () => void }) {
  const t = useTranslations("onboarding");
  const updateStep = useUpdateOnboardingStep();
  const [form, setForm] = useState({
    name: "",
    address: "",
    city: "",
    post_code: "",
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim()) {
      toast.error("Nazwa magazynu jest wymagana");
      return;
    }
    try {
      await apiClient("/v1/warehouses", {
        method: "POST",
        body: JSON.stringify({ ...form, is_default: true }),
      });
      await updateStep.mutateAsync({ step: 2, action: "completed" });
      onNext();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("error"));
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-2 sm:col-span-2">
          <Label htmlFor="wh_name">
            Nazwa magazynu <span className="text-destructive">*</span>
          </Label>
          <Input
            id="wh_name"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            placeholder={t("magazynGłowny")}
          />
        </div>
        <div className="space-y-2 sm:col-span-2">
          <Label htmlFor="wh_address">Ulica i numer</Label>
          <Input
            id="wh_address"
            value={form.address}
            onChange={(e) => setForm({ ...form, address: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="wh_city">Miasto</Label>
          <Input
            id="wh_city"
            value={form.city}
            onChange={(e) => setForm({ ...form, city: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="wh_post_code">Kod pocztowy</Label>
          <Input
            id="wh_post_code"
            value={form.post_code}
            onChange={(e) => setForm({ ...form, post_code: e.target.value })}
            placeholder="00-000"
          />
        </div>
      </div>
      <div className="flex justify-between pt-2">
        <Button type="button" variant="ghost" onClick={onSkip}>
          {t("dismiss")}
        </Button>
        <Button type="submit" disabled={updateStep.isPending}>
          {updateStep.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Dalej
        </Button>
      </div>
    </form>
  );
}

// ---------------------------------------------------------------------------
// Step 3: Integration
// ---------------------------------------------------------------------------
function Step3Integration({ onNext, onSkip }: { onNext: () => void; onSkip: () => void }) {
  const t = useTranslations("onboarding");
  const router = useRouter();
  const updateStep = useUpdateOnboardingStep();
  const [provider, setProvider] = useState("");
  const [apiKey, setApiKey] = useState("");

  const PROVIDERS = [
    { value: "allegro", label: "Allegro", oauth: true },
    { value: "amazon", label: "Amazon", oauth: false },
    { value: "shopify", label: "Shopify", oauth: false },
    { value: "woocommerce", label: "WooCommerce", oauth: false },
    { value: "shoper", label: "Shoper", oauth: false },
    { value: "prestashop", label: "PrestaShop", oauth: false },
  ];

  const selectedProvider = PROVIDERS.find((p) => p.value === provider);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!provider) {
      toast.error(t("form.shipmentProviderPlaceholder"));
      return;
    }
    if (selectedProvider?.oauth) {
      // OAuth providers need full integration flow — mark step and redirect
      await updateStep.mutateAsync({ step: 3, action: "completed" });
      router.push("/integrations");
      return;
    }
    try {
      await apiClient("/v1/integrations", {
        method: "POST",
        body: JSON.stringify({ provider, credentials: { api_key: apiKey } }),
      });
      await updateStep.mutateAsync({ step: 3, action: "completed" });
      onNext();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("error"));
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {PROVIDERS.map((p) => (
          <button
            key={p.value}
            type="button"
            onClick={() => setProvider(p.value)}
            className={`rounded-lg border p-4 text-left text-sm font-medium transition-colors ${
              provider === p.value
                ? "border-primary bg-primary/5"
                : "border-border hover:border-primary/50"
            }`}
          >
            {p.label}
            {p.oauth && (
              <span className="block text-xs text-muted-foreground font-normal mt-1">OAuth</span>
            )}
          </button>
        ))}
      </div>
      {provider && selectedProvider?.oauth && (
        <p className="text-sm text-muted-foreground">
          {t("allegroWymagaAutoryzacjiOauthPoKliknieciuQuotdalej")}
        </p>
      )}
      {provider && !selectedProvider?.oauth && (
        <div className="space-y-2">
          <Label htmlFor="api_key">Klucz API</Label>
          <Input
            id="api_key"
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="Wklej klucz API"
          />
        </div>
      )}
      <div className="flex justify-between pt-2">
        <Button type="button" variant="ghost" onClick={onSkip}>
          {t("dismiss")}
        </Button>
        <Button type="submit" disabled={updateStep.isPending || !provider}>
          {updateStep.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Dalej
        </Button>
      </div>
    </form>
  );
}

// ---------------------------------------------------------------------------
// Step 4: Team
// ---------------------------------------------------------------------------
function Step4Team({ onNext, onSkip }: { onNext: () => void; onSkip: () => void }) {
  const t = useTranslations("onboarding");
  const updateStep = useUpdateOnboardingStep();
  const completeOnboarding = useCompleteOnboarding();
  const [email, setEmail] = useState("");
  const [invited, setInvited] = useState<string[]>([]);
  const [sending, setSending] = useState(false);

  const handleInvite = async () => {
    if (!email.trim()) return;
    setSending(true);
    try {
      await apiClient("/v1/invitations", {
        method: "POST",
        body: JSON.stringify({ email, role: "member" }),
      });
      setInvited((prev) => [...prev, email]);
      setEmail("");
      toast.success(t("zaproszenieWysłane"));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("error"));
    } finally {
      setSending(false);
    }
  };

  const handleFinish = async () => {
    try {
      await updateStep.mutateAsync({ step: 4, action: "completed" });
      await completeOnboarding.mutateAsync();
      onNext();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("error"));
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <Input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="adres@email.pl"
          onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); handleInvite(); } }}
        />
        <Button type="button" variant="outline" onClick={handleInvite} disabled={sending || !email.trim()}>
          {sending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Wyślij"}
        </Button>
      </div>
      {invited.length > 0 && (
        <div className="rounded-md border p-3 space-y-1">
          {invited.map((inv) => (
            <div key={inv} className="flex items-center gap-2 text-sm">
              <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" />
              {inv}
            </div>
          ))}
        </div>
      )}
      <div className="flex justify-between pt-2">
        <Button
          type="button"
          variant="ghost"
          onClick={async () => {
            try {
              await updateStep.mutateAsync({ step: 4, action: "skipped" });
              await completeOnboarding.mutateAsync();
              onSkip();
            } catch (err) {
              toast.error(err instanceof Error ? err.message : t("error"));
            }
          }}
        >
          {t("dismiss")}
        </Button>
        <Button
          type="button"
          onClick={handleFinish}
          disabled={updateStep.isPending || completeOnboarding.isPending}
        >
          {(updateStep.isPending || completeOnboarding.isPending) && (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          )}
          {t("zakonczKonfiguracje")}
        </Button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Completion screen
// ---------------------------------------------------------------------------
function CompletionScreen() {
  const t = useTranslations("onboarding");
  const router = useRouter();
  return (
    <div className="flex flex-col items-center gap-6 py-8 text-center">
      <div className="flex h-20 w-20 items-center justify-center rounded-full bg-green-100">
        <CheckCircle2 className="h-10 w-10 text-green-600" />
      </div>
      <div>
        <h2 className="text-2xl font-bold">Gotowe!</h2>
        <p className="text-muted-foreground mt-1">
          {t("konfiguracjaZakonczonaMozeszTerazKorzystacZSystemu")}
        </p>
      </div>
      <Button size="lg" onClick={() => router.replace("/")}>
        {t("przejdzDoPanelu")}
      </Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------
export default function OnboardingPage() {
  const t = useTranslations("onboarding");
  const router = useRouter();
  const { data: status, isLoading, isError } = useOnboardingStatus();
  const updateStep = useUpdateOnboardingStep();
  const dismissOnboarding = useDismissOnboarding();
  const [localStep, setLocalStep] = useState<number | null>(null);
  const [showCompletion, setShowCompletion] = useState(false);

  useEffect(() => {
    if (!status) return;
    if (status.completed || status.dismissed) {
      router.replace("/");
      return;
    }
    if (localStep === null) {
      setLocalStep(status.current_step);
    }
  }, [status, router, localStep]);

  if (isError) {
    return (
      <div className="flex items-center justify-center py-24 text-center">
        <div className="space-y-2">
          <p className="text-destructive font-medium">{t("nieudałosiezaładowacstatusukonfiguracji")}</p>
          <Button variant="outline" onClick={() => router.replace("/")}>
            {t("wrocDoPanelu")}
          </Button>
        </div>
      </div>
    );
  }

  if (isLoading || localStep === null) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (showCompletion) {
    return <CompletionScreen />;
  }

  const completedSteps = status?.completed_steps ?? [];

  const handleNext = () => {
    const next = localStep + 1;
    if (next > 4) {
      setShowCompletion(true);
    } else {
      setLocalStep(next);
    }
  };

  const handleSkip = async (step: number) => {
    try {
      await updateStep.mutateAsync({ step, action: "skipped" });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("nieudałosiezapisacpominieciakroku"));
      return;
    }
    const next = step + 1;
    if (next > 4) {
      setShowCompletion(true);
    } else {
      setLocalStep(next);
    }
  };

  const handleFinishLater = async () => {
    try {
      await dismissOnboarding.mutateAsync();
    } catch {
      // Still redirect even if dismiss fails — user wants to leave
    }
    router.replace("/");
  };

  const stepTitles: Record<number, string> = {
    1: "Dane firmy",
    2: t("domyslnyMagazyn"),
    3: "Pierwsza integracja",
    4: t("zaproszespoł"),
  };

  const stepDescriptions: Record<number, string> = {
    1: t("uzupełnijdanefirmypotrzebnedofakturiwysyłek"),
    2: t("dodajMagazynZKtoregoBedaRealizowaneZamowienia"),
    3: t("połaczsiezplatformasprzedazowalubkurierem"),
    4: t("zaprosczłonkowzespołudosystemu"),
  };

  return (
    <div className="space-y-2">
      <div className="text-center mb-6">
        <h1 className="text-2xl font-bold">Konfiguracja systemu</h1>
        <p className="text-muted-foreground text-sm mt-1">
          {t("skonfigurujSystemAbyMocWPełniZ")}
        </p>
      </div>

      <Stepper currentStep={localStep} completedSteps={completedSteps} />

      <Card>
        <CardHeader>
          <CardTitle>{stepTitles[localStep]}</CardTitle>
          <CardDescription>{stepDescriptions[localStep]}</CardDescription>
        </CardHeader>
        <CardContent>
          {localStep === 1 && <Step1Company onNext={handleNext} />}
          {localStep === 2 && (
            <Step2Warehouse
              onNext={handleNext}
              onSkip={() => handleSkip(2)}
            />
          )}
          {localStep === 3 && (
            <Step3Integration
              onNext={handleNext}
              onSkip={() => handleSkip(3)}
            />
          )}
          {localStep === 4 && (
            <Step4Team
              onNext={() => setShowCompletion(true)}
              onSkip={() => setShowCompletion(true)}
            />
          )}
        </CardContent>
      </Card>

      <div className="flex justify-center pt-4">
        <Button variant="ghost" size="sm" onClick={handleFinishLater}>
          {t("dokonczPozniej")}
        </Button>
      </div>
    </div>
  );
}
