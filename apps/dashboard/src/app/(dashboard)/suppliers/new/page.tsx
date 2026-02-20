"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import Image from "next/image";
import {
  ArrowLeft,
  ChevronDown,
  Eye,
  EyeOff,
  FileText,
  Info,
  KeyRound,
  Loader2,
} from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useCreateSupplier } from "@/hooks/use-suppliers";
import { useCreateIntegration } from "@/hooks/use-integrations";
import { getErrorMessage } from "@/lib/api-client";
import { cn } from "@/lib/utils";
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

const fileSupplierSchema = z.object({
  name: z.string().min(1, "Nazwa jest wymagana"),
  code: z.string().optional(),
  feed_url: z.string().optional(),
  feed_format: z.string().optional(),
  sync_interval_minutes: z.number().min(5).max(1440).optional(),
});

type FileSupplierForm = z.infer<typeof fileSupplierSchema>;

type Step = "pick" | "file" | "btp";

// ---------------------------------------------------------------------------
// Provider cards for the picker
// ---------------------------------------------------------------------------

const providers = [
  {
    key: "btp" as const,
    logo: "/logos/btp.svg",
    name: "BTP.pro",
    description: "Integracja API z hurtownią B2B na platformie BTP",
  },
  {
    key: "file" as const,
    icon: FileText,
    name: "Plik / URL",
    description: "Import produktów z pliku IOF, CSV lub zewnętrznego URL",
  },
];

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function NewSupplierPage() {
  const router = useRouter();
  const [step, setStep] = useState<Step>("pick");

  // --- Shared mutations ---
  const createIntegration = useCreateIntegration();

  // --- File/URL form ---
  const createSupplier = useCreateSupplier();
  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<FileSupplierForm>({
    resolver: zodResolver(fileSupplierSchema),
    defaultValues: { feed_format: "iof", sync_interval_minutes: 60 },
  });

  const selectedFormat = watch("feed_format");

  // BTP API credentials for XML hybrid mode
  const [xmlApiExpanded, setXmlApiExpanded] = useState(false);
  const [xmlPublicKey, setXmlPublicKey] = useState("");
  const [xmlPrivateKey, setXmlPrivateKey] = useState("");
  const [xmlApiBaseUrl, setXmlApiBaseUrl] = useState("");
  const [showXmlPrivateKey, setShowXmlPrivateKey] = useState(false);
  const [fileSubmitting, setFileSubmitting] = useState(false);

  const onFileSubmit = async (data: FileSupplierForm) => {
    const hasBtpCreds =
      selectedFormat === "xml" &&
      xmlPublicKey.trim() &&
      xmlPrivateKey.trim();

    if (!hasBtpCreds) {
      createSupplier.mutate(data, {
        onSuccess: () => {
          toast.success("Dostawca został utworzony");
          router.push("/suppliers");
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      });
      return;
    }

    // Hybrid mode: create integration first, then supplier with integration_id
    setFileSubmitting(true);
    try {
      const integration = await createIntegration.mutateAsync({
        provider: "btp",
        label: data.name,
        credentials: {
          username: xmlPublicKey,
          password: xmlPrivateKey,
          base_url: xmlApiBaseUrl || undefined,
        },
      });

      await createSupplier.mutateAsync({
        ...data,
        integration_id: integration.id,
      });

      toast.success("Dostawca został utworzony (tryb hybrydowy XML + API)");
      router.push("/suppliers");
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setFileSubmitting(false);
    }
  };

  // --- BTP form ---
  const createSupplierApi = useCreateSupplier();
  const [btpName, setBtpName] = useState("");
  const [btpPublicKey, setBtpPublicKey] = useState("");
  const [btpPrivateKey, setBtpPrivateKey] = useState("");
  const [btpBaseUrl, setBtpBaseUrl] = useState("");
  const [showPrivateKey, setShowPrivateKey] = useState(false);
  const [btpSubmitting, setBtpSubmitting] = useState(false);

  const onBtpSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!btpName.trim()) {
      toast.error("Nazwa dostawcy jest wymagana");
      return;
    }
    if (!btpPublicKey.trim()) {
      toast.error("Klucz publiczny jest wymagany");
      return;
    }
    if (!btpPrivateKey.trim()) {
      toast.error("Klucz prywatny jest wymagany");
      return;
    }

    setBtpSubmitting(true);
    try {
      const integration = await createIntegration.mutateAsync({
        provider: "btp",
        label: btpName,
        credentials: {
          username: btpPublicKey,
          password: btpPrivateKey,
          base_url: btpBaseUrl || undefined,
        },
      });

      await createSupplierApi.mutateAsync({
        name: btpName,
        feed_format: "btp",
        integration_id: integration.id,
      });

      toast.success("Dostawca BTP został utworzony");
      router.push("/suppliers");
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setBtpSubmitting(false);
    }
  };

  // --- Navigation ---
  const handleBack = () => {
    if (step === "pick") {
      router.push("/suppliers");
    } else {
      setStep("pick");
    }
  };

  const titles: Record<Step, string> = {
    pick: "Nowy dostawca",
    file: "Nowy dostawca — Plik / URL",
    btp: "Nowy dostawca — BTP.pro",
  };

  return (
    <AdminGuard>
      <div className="mx-auto max-w-3xl">
        <div className="flex items-center gap-4 mb-6">
          <Button variant="ghost" size="icon" onClick={handleBack}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="text-2xl font-bold tracking-tight">
            {titles[step]}
          </h1>
        </div>

        {/* ── Step 0: Provider picker ── */}
        {step === "pick" && (
          <div className="grid gap-4 sm:grid-cols-2">
            {providers.map((p) => (
              <button
                key={p.key}
                type="button"
                onClick={() => setStep(p.key === "btp" ? "btp" : "file")}
                className="flex items-start gap-4 rounded-xl border p-4 text-left transition-colors hover:bg-muted/50"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                  {"logo" in p && p.logo ? (
                    <Image
                      src={p.logo}
                      alt={p.name}
                      width={24}
                      height={24}
                      className="h-6 w-6 object-contain"
                    />
                  ) : (
                    "icon" in p && p.icon && (
                      <p.icon className="h-5 w-5 text-muted-foreground" />
                    )
                  )}
                </div>
                <div className="min-w-0">
                  <p className="font-medium">{p.name}</p>
                  <p className="text-sm text-muted-foreground">
                    {p.description}
                  </p>
                </div>
              </button>
            ))}
          </div>
        )}

        {/* ── Step 1a: File / URL form ── */}
        {step === "file" && (
          <Card>
            <CardHeader>
              <CardTitle>Dane dostawcy</CardTitle>
              <CardDescription>
                Dodaj dostawcę importującego produkty z pliku lub URL
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
                  <Label htmlFor="feed_url">URL feeda produktów</Label>
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
                      Interwał synchronizacji
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
                        <SelectItem value="15">Co 15 minut</SelectItem>
                        <SelectItem value="30">Co 30 minut</SelectItem>
                        <SelectItem value="60">Co 1 godzinę</SelectItem>
                        <SelectItem value="120">Co 2 godziny</SelectItem>
                        <SelectItem value="360">Co 6 godzin</SelectItem>
                        <SelectItem value="720">Co 12 godzin</SelectItem>
                        <SelectItem value="1440">Raz dziennie</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                {/* BTP API credentials for XML hybrid mode */}
                {selectedFormat === "xml" && (
                  <div className="rounded-lg border">
                    <button
                      type="button"
                      className="flex w-full items-center justify-between p-3 text-sm font-medium hover:bg-muted/50 transition-colors"
                      onClick={() => setXmlApiExpanded((prev) => !prev)}
                    >
                      <div className="flex items-center gap-2">
                        <KeyRound className="h-4 w-4 text-muted-foreground" />
                        Klucze API BTP (opcjonalnie)
                      </div>
                      <ChevronDown
                        className={cn(
                          "h-4 w-4 text-muted-foreground transition-transform",
                          xmlApiExpanded && "rotate-180"
                        )}
                      />
                    </button>
                    {xmlApiExpanded && (
                      <div className="border-t px-3 pb-3 pt-3 space-y-3">
                        <p className="text-xs text-muted-foreground">
                          Podaj klucze API z panelu BTP, aby włączyć tryb
                          hybrydowy — pełny katalog z XML + aktualizacje stanów
                          przez API między synchronizacjami.
                        </p>
                        <div className="space-y-2">
                          <Label htmlFor="xml-public-key">
                            Klucz publiczny (login)
                          </Label>
                          <Input
                            id="xml-public-key"
                            value={xmlPublicKey}
                            onChange={(e) => setXmlPublicKey(e.target.value)}
                            placeholder="Klucz publiczny z panelu BTP"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="xml-private-key">
                            Klucz prywatny (hasło)
                          </Label>
                          <div className="relative">
                            <Input
                              id="xml-private-key"
                              type={showXmlPrivateKey ? "text" : "password"}
                              className="pr-10"
                              value={xmlPrivateKey}
                              onChange={(e) => setXmlPrivateKey(e.target.value)}
                              placeholder="Klucz prywatny z panelu BTP"
                            />
                            <button
                              type="button"
                              className="absolute right-0 top-0 flex h-9 w-9 items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                              onClick={() =>
                                setShowXmlPrivateKey((prev) => !prev)
                              }
                              tabIndex={-1}
                              aria-label={
                                showXmlPrivateKey
                                  ? "Ukryj klucz"
                                  : "Pokaż klucz"
                              }
                            >
                              {showXmlPrivateKey ? (
                                <EyeOff className="h-4 w-4" />
                              ) : (
                                <Eye className="h-4 w-4" />
                              )}
                            </button>
                          </div>
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="xml-base-url">
                            Adres API hurtowni{" "}
                            <span className="text-muted-foreground font-normal">
                              (opcjonalnie)
                            </span>
                          </Label>
                          <Input
                            id="xml-base-url"
                            type="url"
                            value={xmlApiBaseUrl}
                            onChange={(e) => setXmlApiBaseUrl(e.target.value)}
                            placeholder="https://twoja-hurtownia.btp.pro"
                          />
                        </div>
                      </div>
                    )}
                  </div>
                )}
                <div className="flex gap-2 pt-2">
                  <Button
                    type="submit"
                    disabled={createSupplier.isPending || fileSubmitting}
                  >
                    {createSupplier.isPending || fileSubmitting ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Tworzenie...
                      </>
                    ) : (
                      "Utwórz dostawcę"
                    )}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setStep("pick")}
                  >
                    Wstecz
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        )}

        {/* ── Step 1b: BTP.pro ── */}
        {step === "btp" && (
          <div className="space-y-4">
            {/* Connection instructions */}
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Info className="h-4 w-4 text-muted-foreground" />
                  <CardTitle className="text-base">Jak się połączyć</CardTitle>
                </div>
              </CardHeader>
              <CardContent className="space-y-3 text-sm text-muted-foreground">
                <p>
                  BTP.pro udostępnia API oparte o uwierzytelnianie{" "}
                  <span className="font-medium text-foreground">Basic Auth</span>{" "}
                  z parą kluczy: publicznym (login) i prywatnym (hasło).
                </p>
                <div className="space-y-2">
                  <p className="font-medium text-foreground">
                    Aby uzyskać klucze:
                  </p>
                  <ol className="list-decimal list-inside space-y-1 pl-1">
                    <li>Zaloguj się do panelu B2B swojej hurtowni</li>
                    <li>
                      Przejdź do{" "}
                      <span className="font-medium text-foreground">
                        Profil użytkownika → zakładka Klucze
                      </span>
                    </li>
                    <li>Skopiuj klucz publiczny i prywatny</li>
                  </ol>
                  <p className="text-xs">
                    Jeśli zakładka Klucze nie jest widoczna, skontaktuj się z
                    opiekunem handlowym — wymagane jest uprawnienie ClientApi.
                  </p>
                </div>
                <div className="rounded-lg border bg-muted/30 p-3 text-xs">
                  <div className="flex items-center gap-1.5 font-medium text-foreground mb-1">
                    <KeyRound className="h-3.5 w-3.5" />
                    Zalecane częstotliwości synchronizacji
                  </div>
                  <ul className="space-y-0.5 pl-5 list-disc">
                    <li>Katalog produktów: 1–2 razy dziennie</li>
                    <li>Stany magazynowe: 4–24 razy dziennie</li>
                  </ul>
                </div>
              </CardContent>
            </Card>

            {/* Credential form */}
            <Card>
              <CardHeader>
                <CardTitle>Dane połączenia</CardTitle>
                <CardDescription>
                  Podaj klucze API z panelu B2B hurtowni
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={onBtpSubmit} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="btp-name">Nazwa dostawcy *</Label>
                    <Input
                      id="btp-name"
                      value={btpName}
                      onChange={(e) => setBtpName(e.target.value)}
                      placeholder="np. Hurtownia XYZ (BTP)"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="btp-public-key">
                      Klucz publiczny (login) *
                    </Label>
                    <Input
                      id="btp-public-key"
                      value={btpPublicKey}
                      onChange={(e) => setBtpPublicKey(e.target.value)}
                      placeholder="Twój klucz publiczny z panelu BTP"
                    />
                    <p className="text-xs text-muted-foreground">
                      Profil użytkownika → zakładka Klucze
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="btp-private-key">
                      Klucz prywatny (hasło) *
                    </Label>
                    <div className="relative">
                      <Input
                        id="btp-private-key"
                        type={showPrivateKey ? "text" : "password"}
                        className="pr-10"
                        value={btpPrivateKey}
                        onChange={(e) => setBtpPrivateKey(e.target.value)}
                        placeholder="Twój klucz prywatny z panelu BTP"
                      />
                      <button
                        type="button"
                        className="absolute right-0 top-0 flex h-9 w-9 items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                        onClick={() => setShowPrivateKey((prev) => !prev)}
                        tabIndex={-1}
                        aria-label={
                          showPrivateKey ? "Ukryj klucz" : "Pokaż klucz"
                        }
                      >
                        {showPrivateKey ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </button>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="btp-base-url">
                      Adres API hurtowni{" "}
                      <span className="text-muted-foreground font-normal">
                        (opcjonalnie)
                      </span>
                    </Label>
                    <Input
                      id="btp-base-url"
                      type="url"
                      value={btpBaseUrl}
                      onChange={(e) => setBtpBaseUrl(e.target.value)}
                      placeholder="https://twoja-hurtownia.btp.pro"
                    />
                    <p className="text-xs text-muted-foreground">
                      Adres Swagger API hurtowni. Pozostaw puste jeśli nie
                      znasz — skonfigurujemy automatycznie.
                    </p>
                  </div>
                  <div className="flex gap-2 pt-2">
                    <Button type="submit" disabled={btpSubmitting}>
                      {btpSubmitting ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Łączenie...
                        </>
                      ) : (
                        "Połącz i utwórz dostawcę"
                      )}
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => setStep("pick")}
                      disabled={btpSubmitting}
                    >
                      Wstecz
                    </Button>
                  </div>
                </form>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </AdminGuard>
  );
}
