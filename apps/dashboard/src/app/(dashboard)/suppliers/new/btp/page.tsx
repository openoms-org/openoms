"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  ArrowLeft,
  Check,
  Eye,
  EyeOff,
  FileText,
  Info,
  KeyRound,
  Loader2,
  Settings,
  AlertCircle,
  RotateCw,
} from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useBTPWizardStartImport,
  useBTPWizardImportProgress,
  useBTPWizardSetAPIKeys,
  useBTPWizardCompleteSyncSettings,
} from "@/hooks/use-suppliers";
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
import { Progress } from "@/components/ui/progress";
import { useTranslations } from "next-intl";

// ---------------------------------------------------------------------------
// Steps
// ---------------------------------------------------------------------------

type WizardStep = 1 | 2 | 3;

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function BTPWizardPage() {
  const t = useTranslations("suppliers");
  const tc = useTranslations("common");
  const router = useRouter();
  const [currentStep, setCurrentStep] = useState<WizardStep>(1);

  const steps = [
    { num: 1 as const, label: t("btp.step1"), icon: FileText },
    { num: 2 as const, label: t("btp.step2"), icon: KeyRound },
    { num: 3 as const, label: t("btp.step3"), icon: Settings },
  ];
  const [supplierId, setSupplierId] = useState<string | null>(null);

  // ── Step 1 state ──
  const [name, setName] = useState("");
  const [feedUrl, setFeedUrl] = useState("");
  const [importStarted, setImportStarted] = useState(false);

  const startImport = useBTPWizardStartImport();
  const {
    data: progress,
    isError: progressError,
  } = useBTPWizardImportProgress(supplierId ?? "", importStarted && !!supplierId);

  const importDone = progress?.status === "completed";
  const importFailed = progress?.status === "failed";
  const importRunning = progress?.status === "running" || progress?.status === "pending";
  const importPercent =
    progress && progress.total > 0
      ? Math.round((progress.processed / progress.total) * 100)
      : 0;

  const onStartImport = () => {
    if (!name.trim()) {
      toast.error("Nazwa dostawcy jest wymagana");
      return;
    }
    if (!feedUrl.trim()) {
      toast.error("URL pliku XML jest wymagany");
      return;
    }

    startImport.mutate(
      { name: name.trim(), feed_url: feedUrl.trim() },
      {
        onSuccess: (supplier) => {
          setSupplierId(supplier.id);
          setImportStarted(true);
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  const onRetryImport = () => {
    setImportStarted(false);
    setSupplierId(null);
    // User can click "Rozpocznij import" again
  };

  // ── Step 2 state ──
  const [publicKey, setPublicKey] = useState("");
  const [privateKey, setPrivateKey] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [showPrivateKey, setShowPrivateKey] = useState(false);

  const setAPIKeys = useBTPWizardSetAPIKeys(supplierId ?? "");

  const onSubmitAPIKeys = (e: React.FormEvent) => {
    e.preventDefault();
    if (!publicKey.trim()) {
      toast.error("Klucz publiczny jest wymagany");
      return;
    }
    if (!privateKey.trim()) {
      toast.error("Klucz prywatny jest wymagany");
      return;
    }

    setAPIKeys.mutate(
      {
        public_key: publicKey.trim(),
        private_key: privateKey.trim(),
        base_url: baseUrl.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast.success("Klucze API zweryfikowane");
          setCurrentStep(3);
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  // ── Step 3 state ──
  const [xmlInterval, setXmlInterval] = useState("24");
  const [apiInterval, setApiInterval] = useState("30");

  const completeSyncSettings = useBTPWizardCompleteSyncSettings(supplierId ?? "");

  const onCompleteSyncSettings = () => {
    completeSyncSettings.mutate(
      {
        xml_sync_interval_hours: parseInt(xmlInterval, 10),
        api_sync_interval_minutes: parseInt(apiInterval, 10),
      },
      {
        onSuccess: () => {
          toast.success("Dostawca BTP skonfigurowany");
          router.push(`/suppliers/${supplierId}`);
        },
        onError: (error) => toast.error(getErrorMessage(error)),
      }
    );
  };

  return (
    <AdminGuard>
      <div className="mx-auto max-w-3xl">
        {/* Header */}
        <div className="flex items-center gap-4 mb-6">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => router.push("/suppliers/new")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="text-2xl font-bold tracking-tight">
            Nowy dostawca — BTP.pro
          </h1>
        </div>

        {/* Step indicator */}
        <div className="flex items-center gap-2 mb-8">
          {steps.map((s, idx) => (
            <div key={s.num} className="flex items-center gap-2">
              {idx > 0 && (
                <div
                  className={cn(
                    "h-px w-8 sm:w-12",
                    currentStep > s.num - 1 ? "bg-primary" : "bg-border"
                  )}
                />
              )}
              <div
                className={cn(
                  "flex items-center gap-2 rounded-full px-3 py-1.5 text-sm font-medium transition-colors",
                  currentStep === s.num
                    ? "bg-primary text-primary-foreground"
                    : currentStep > s.num
                      ? "bg-primary/10 text-primary"
                      : "bg-muted text-muted-foreground"
                )}
              >
                {currentStep > s.num ? (
                  <Check className="h-3.5 w-3.5" />
                ) : (
                  <s.icon className="h-3.5 w-3.5" />
                )}
                <span className="hidden sm:inline">{s.label}</span>
                <span className="sm:hidden">{s.num}</span>
              </div>
            </div>
          ))}
        </div>

        {/* ── Step 1: Import XML ── */}
        {currentStep === 1 && (
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Info className="h-4 w-4 text-muted-foreground" />
                  <CardTitle className="text-base">
                    {t("gdzieZnalezcLinkDoPlikuXml")}
                  </CardTitle>
                </div>
              </CardHeader>
              <CardContent className="space-y-2 text-sm text-muted-foreground">
                <ol className="list-decimal list-inside space-y-1 pl-1">
                  <li>{t("zalogujSieDoPaneluB2bHurtowni")}</li>
                  <li>
                    W menu bocznym kliknij{" "}
                    <span className="font-medium text-foreground">
                      Eksport danych XML
                    </span>
                  </li>
                  <li>
                    Znajdź pozycję{" "}
                    <span className="font-medium text-foreground">
                      Plik w formacie Comarch EDI
                    </span>
                  </li>
                  <li>
                    Skopiuj link z kolumny{" "}
                    <span className="font-medium text-foreground">
                      Link do pobierania automatycznego
                    </span>
                  </li>
                </ol>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>{t("importKataloguProduktow")}</CardTitle>
                <CardDescription>
                  {t("podajNazweDostawcyILinkDoPliku")}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="btp-name">Nazwa dostawcy *</Label>
                  <Input
                    id="btp-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="np. Hurtownia XYZ"
                    disabled={importStarted}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="btp-feed-url">URL pliku XML *</Label>
                  <Input
                    id="btp-feed-url"
                    value={feedUrl}
                    onChange={(e) => setFeedUrl(e.target.value)}
                    placeholder="https://hurtownia.btp.pro/integration/xml/..."
                    disabled={importStarted}
                  />
                </div>

                {/* Import progress */}
                {importStarted && (
                  <div className="space-y-3 pt-2">
                    {importRunning && (
                      <>
                        <div className="flex items-center justify-between text-sm">
                          <span className="text-muted-foreground">
                            {t("importowanieProduktow")}
                          </span>
                          <span className="font-medium tabular-nums">
                            {progress
                              ? `${progress.processed.toLocaleString("pl-PL")} / ${progress.total.toLocaleString("pl-PL")}`
                              : "Rozpoczynanie..."}
                          </span>
                        </div>
                        <Progress value={importPercent} />
                        <p className="text-xs text-muted-foreground">
                          {importPercent}% — nie zamykaj tej strony
                        </p>
                      </>
                    )}

                    {importDone && (
                      <div className="flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-800 dark:border-green-800 dark:bg-green-950/30 dark:text-green-200">
                        <Check className="h-4 w-4 shrink-0" />
                        <span>
                          Import zakończony —{" "}
                          {progress.processed.toLocaleString("pl-PL")} produktów
                          zaimportowanych
                        </span>
                      </div>
                    )}

                    {(importFailed || progressError) && (
                      <div className="flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
                        <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
                        <div className="space-y-1">
                          <p>{t("importniepowiodłsie")}</p>
                          {progress?.error && (
                            <p className="text-xs opacity-80">
                              {progress.error}
                            </p>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                )}

                <div className="flex gap-2 pt-2">
                  {!importStarted && (
                    <Button
                      onClick={onStartImport}
                      disabled={startImport.isPending}
                    >
                      {startImport.isPending ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Rozpoczynanie...
                        </>
                      ) : (
                        "Rozpocznij import"
                      )}
                    </Button>
                  )}
                  {importDone && (
                    <Button onClick={() => setCurrentStep(2)}>Dalej</Button>
                  )}
                  {(importFailed || progressError) && (
                    <Button variant="outline" onClick={onRetryImport}>
                      <RotateCw className="mr-2 h-4 w-4" />
                      {t("retry")}
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          </div>
        )}

        {/* ── Step 2: API Keys ── */}
        {currentStep === 2 && (
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Info className="h-4 w-4 text-muted-foreground" />
                  <CardTitle className="text-base">
                    {t("gdzieZnalezcKluczeApi")}
                  </CardTitle>
                </div>
              </CardHeader>
              <CardContent className="space-y-2 text-sm text-muted-foreground">
                <ol className="list-decimal list-inside space-y-1 pl-1">
                  <li>{t("zalogujSieDoPaneluB2bHurtowni")}</li>
                  <li>
                    Przejdź do{" "}
                    <span className="font-medium text-foreground">
                      {t("mojProfilZakładkaKlucze")}
                    </span>
                  </li>
                  <li>{t("skopiujkluczpublicznyloginiprywatnyhasło")}</li>
                </ol>
                <p className="text-xs">
                  {t("jesliZakładkaKluczeNieJestWidocznaSkontaktuj")}
                  opiekunem handlowym — wymagane jest uprawnienie ClientApi.
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Klucze API</CardTitle>
                <CardDescription>
                  {t("podajKluczeDoSynchronizacjiStanowMagazynowychI")}
                  czasie rzeczywistym
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={onSubmitAPIKeys} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="btp-public-key">
                      Klucz publiczny (login) *
                    </Label>
                    <Input
                      id="btp-public-key"
                      value={publicKey}
                      onChange={(e) => setPublicKey(e.target.value)}
                      placeholder={t("btp.usernamePlaceholder")}
                    />
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
                        value={privateKey}
                        onChange={(e) => setPrivateKey(e.target.value)}
                        placeholder={t("btp.passwordPlaceholder")}
                      />
                      <button
                        type="button"
                        className="absolute right-0 top-0 flex h-9 w-9 items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                        onClick={() => setShowPrivateKey((prev) => !prev)}
                        tabIndex={-1}
                        aria-label={
                          showPrivateKey ? "Ukryj klucz" : t("pokazKlucz")
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
                      value={baseUrl}
                      onChange={(e) => setBaseUrl(e.target.value)}
                      placeholder="https://twoja-hurtownia.btp.pro"
                    />
                    <p className="text-xs text-muted-foreground">
                      {t("pozostawPusteJesliNieZnaszSkonfigurujemy")}
                      automatycznie.
                    </p>
                  </div>
                  <div className="flex gap-2 pt-2">
                    <Button type="submit" disabled={setAPIKeys.isPending}>
                      {setAPIKeys.isPending ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Weryfikowanie...
                        </>
                      ) : (
                        "Weryfikuj i zapisz"
                      )}
                    </Button>
                  </div>
                </form>
              </CardContent>
            </Card>
          </div>
        )}

        {/* ── Step 3: Sync settings ── */}
        {currentStep === 3 && (
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Info className="h-4 w-4 text-muted-foreground" />
                  <CardTitle className="text-base">
                    {t("jakDziałaSynchronizacja")}
                  </CardTitle>
                </div>
              </CardHeader>
              <CardContent className="space-y-3 text-sm text-muted-foreground">
                <p>
                  {t("btpWykorzystujeDwaZrodłaDanychKtoreSynchronizuja")}
                  {t("niezaleznie")}
                </p>
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="rounded-lg border p-3 space-y-1">
                    <div className="flex items-center gap-1.5 font-medium text-foreground">
                      <FileText className="h-3.5 w-3.5" />
                      Katalog XML
                    </div>
                    <p className="text-xs">
                      {t("pełnaListaProduktowDodawanieIUsuwanieProdukty")}
                      {t("nieobecneWXmlZostanaUsunieteAIch")}
                      wyzerowany.
                    </p>
                  </div>
                  <div className="rounded-lg border p-3 space-y-1">
                    <div className="flex items-center gap-1.5 font-medium text-foreground">
                      <KeyRound className="h-3.5 w-3.5" />
                      API (stany i ceny)
                    </div>
                    <p className="text-xs">
                      Aktualne stany magazynowe i ceny w czasie rzeczywistym.
                      {t("daneZApiMajaZawszePriorytetXml")}
                      {t("stanow")}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Ustawienia synchronizacji</CardTitle>
                <CardDescription>
                  {t("ustawCzestotliwoscOdswiezaniaKataloguIStanow")}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>{t("odswiezanieKataloguXml")}</Label>
                  <Select value={xmlInterval} onValueChange={setXmlInterval}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="12">Co 12 godzin</SelectItem>
                      <SelectItem value="24">Co 24 godziny</SelectItem>
                      <SelectItem value="48">Co 48 godzin</SelectItem>
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">
                    {t("jakCzestoSprawdzacNoweIUsunieteProdukty")}
                  </p>
                </div>
                <div className="space-y-2">
                  <Label>{t("synchronizacjaStanowApi")}</Label>
                  <Select value={apiInterval} onValueChange={setApiInterval}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="15">Co 15 minut</SelectItem>
                      <SelectItem value="30">Co 30 minut</SelectItem>
                      <SelectItem value="60">{t("co1Godzine")}</SelectItem>
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">
                    {t("jakCzestoAktualizowacStanyMagazynoweICeny")}
                  </p>
                </div>
                <div className="flex gap-2 pt-2">
                  <Button
                    onClick={onCompleteSyncSettings}
                    disabled={completeSyncSettings.isPending}
                  >
                    {completeSyncSettings.isPending ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Zapisywanie...
                      </>
                    ) : (
                      t("zakonczKonfiguracje")
                    )}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </AdminGuard>
  );
}
