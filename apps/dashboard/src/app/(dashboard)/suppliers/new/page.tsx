"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { ArrowLeft, Loader2 } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { IntegrationMethodSelector } from "@/components/suppliers/integration-method-selector";
import { useCreateSupplier } from "@/hooks/use-suppliers";
import { useCreateIntegration } from "@/hooks/use-integrations";
import { getErrorMessage } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

// ---------------------------------------------------------------------------
// Schemas
// ---------------------------------------------------------------------------

const supplierSchema = z.object({
  name: z.string().min(1, "Nazwa jest wymagana"),
  code: z.string().optional(),
  feed_url: z.string().optional(),
  feed_format: z.string().optional(),
  sync_interval_minutes: z.number().min(5).max(1440).optional(),
});

type SupplierForm = z.infer<typeof supplierSchema>;

type Step = "method" | "file" | "api";

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function NewSupplierPage() {
  const router = useRouter();
  const [step, setStep] = useState<Step>("method");
  const [method, setMethod] = useState<"file" | "api" | null>(null);

  // --- File/URL form ---
  const createSupplier = useCreateSupplier();
  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<SupplierForm>({
    resolver: zodResolver(supplierSchema),
    defaultValues: { feed_format: "iof", sync_interval_minutes: 60 },
  });

  const onFileSubmit = (data: SupplierForm) => {
    createSupplier.mutate(data, {
      onSuccess: () => {
        toast.success("Dostawca zostal utworzony");
        router.push("/suppliers");
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  // --- API (BTP) form ---
  const createIntegration = useCreateIntegration();
  const createSupplierApi = useCreateSupplier();
  const [btpName, setBtpName] = useState("");
  const [btpApiKey, setBtpApiKey] = useState("");
  const [btpApiUrl, setBtpApiUrl] = useState("https://api.btp.pro");
  const [btpSubmitting, setBtpSubmitting] = useState(false);

  const onApiSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!btpName.trim()) {
      toast.error("Nazwa dostawcy jest wymagana");
      return;
    }
    if (!btpApiKey.trim()) {
      toast.error("Klucz API jest wymagany");
      return;
    }

    setBtpSubmitting(true);
    try {
      const integration = await createIntegration.mutateAsync({
        provider: "btp",
        label: btpName,
        credentials: {
          api_key: btpApiKey,
          api_url: btpApiUrl || "https://api.btp.pro",
        },
      });

      await createSupplierApi.mutateAsync({
        name: btpName,
        feed_format: "btp",
        integration_id: integration.id,
      });

      toast.success("Dostawca BTP zostal utworzony");
      router.push("/suppliers");
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setBtpSubmitting(false);
    }
  };

  // --- Navigation ---
  const handleBack = () => {
    if (step === "method") {
      router.push("/suppliers");
    } else {
      setStep("method");
    }
  };

  const handleMethodSelect = (m: "file" | "api") => {
    setMethod(m);
    setStep(m);
  };

  // --- Titles ---
  const titles: Record<Step, string> = {
    method: "Nowy dostawca",
    file: "Nowy dostawca — Plik / URL",
    api: "Nowy dostawca — API (BTP.pro)",
  };

  return (
    <AdminGuard>
      <div className="mx-auto max-w-4xl">
        <div className="flex items-center gap-4 mb-6">
          <Button variant="ghost" size="icon" onClick={handleBack}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="text-2xl font-bold tracking-tight">
            {titles[step]}
          </h1>
        </div>

        {/* ── Step 0: Method selection ── */}
        {step === "method" && (
          <Card>
            <CardHeader>
              <CardTitle>Wybierz metode integracji</CardTitle>
              <CardDescription>
                W jaki sposob chcesz pobierac produkty od dostawcy?
              </CardDescription>
            </CardHeader>
            <CardContent>
              <IntegrationMethodSelector
                selected={method}
                onSelect={handleMethodSelect}
              />
            </CardContent>
          </Card>
        )}

        {/* ── Step 1a: File / URL form ── */}
        {step === "file" && (
          <Card>
            <CardHeader>
              <CardTitle>Dane dostawcy</CardTitle>
              <CardDescription>
                Dodaj dostawce importujacego produkty z pliku lub URL
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form
                onSubmit={handleSubmit(onFileSubmit)}
                className="space-y-4"
              >
                <div className="space-y-2">
                  <Label htmlFor="name">Nazwa *</Label>
                  <Input
                    id="name"
                    {...register("name")}
                    placeholder="np. Hurtownia ABC"
                  />
                  {errors.name && (
                    <p className="text-sm text-destructive">
                      {errors.name.message}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="code">Kod dostawcy</Label>
                  <Input
                    id="code"
                    {...register("code")}
                    placeholder="np. ABC123"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="feed_url">URL feeda produktow</Label>
                  <Input
                    id="feed_url"
                    {...register("feed_url")}
                    placeholder="https://example.com/feed.xml"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Format feeda</Label>
                    <Select
                      defaultValue="iof"
                      onValueChange={(v) => setValue("feed_format", v)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="iof">
                          IOF (Internet Offer Format)
                        </SelectItem>
                        <SelectItem value="csv">CSV</SelectItem>
                        <SelectItem value="xml">XML</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="sync_interval_minutes">
                      Interwal synchronizacji (min)
                    </Label>
                    <Select
                      defaultValue="60"
                      onValueChange={(v) =>
                        setValue("sync_interval_minutes", parseInt(v, 10))
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="5">Co 5 minut</SelectItem>
                        <SelectItem value="15">Co 15 minut</SelectItem>
                        <SelectItem value="30">Co 30 minut</SelectItem>
                        <SelectItem value="60">Co 1 godzine</SelectItem>
                        <SelectItem value="120">Co 2 godziny</SelectItem>
                        <SelectItem value="360">Co 6 godzin</SelectItem>
                        <SelectItem value="720">Co 12 godzin</SelectItem>
                        <SelectItem value="1440">Raz dziennie</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div className="flex gap-2 pt-4">
                  <Button type="submit" disabled={createSupplier.isPending}>
                    {createSupplier.isPending ? "Tworzenie..." : "Utworz dostawce"}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setStep("method")}
                  >
                    Wstecz
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        )}

        {/* ── Step 1b: API (BTP) form ── */}
        {step === "api" && (
          <Card>
            <CardHeader>
              <CardTitle>Konfiguracja BTP.pro</CardTitle>
              <CardDescription>
                Podaj dane dostepowe do API hurtowni BTP
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={onApiSubmit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="btp-name">Nazwa dostawcy *</Label>
                  <Input
                    id="btp-name"
                    value={btpName}
                    onChange={(e) => setBtpName(e.target.value)}
                    placeholder="np. BTP Hurtownia XYZ"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="btp-api-key">Klucz API *</Label>
                  <Input
                    id="btp-api-key"
                    type="password"
                    value={btpApiKey}
                    onChange={(e) => setBtpApiKey(e.target.value)}
                    placeholder="Wklej klucz API z panelu BTP"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="btp-api-url">URL API</Label>
                  <Input
                    id="btp-api-url"
                    value={btpApiUrl}
                    onChange={(e) => setBtpApiUrl(e.target.value)}
                    placeholder="https://api.btp.pro"
                  />
                  <p className="text-xs text-muted-foreground">
                    Domyslnie: https://api.btp.pro
                  </p>
                </div>
                <div className="flex gap-2 pt-4">
                  <Button type="submit" disabled={btpSubmitting}>
                    {btpSubmitting ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Konfiguracja...
                      </>
                    ) : (
                      "Utworz dostawce"
                    )}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setStep("method")}
                    disabled={btpSubmitting}
                  >
                    Wstecz
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        )}
      </div>
    </AdminGuard>
  );
}
