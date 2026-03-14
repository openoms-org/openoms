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
import { useTranslations } from "next-intl";

// ---------------------------------------------------------------------------
// Schemas
// ---------------------------------------------------------------------------

const fileSupplierSchema = z.object({
  name: z.string().min(1),
  code: z.string().optional(),
  feed_url: z.string().optional(),
  feed_format: z.string().optional(),
  sync_interval_minutes: z.number().min(5).max(1440).optional(),
});

type FileSupplierForm = z.infer<typeof fileSupplierSchema>;

type Step = "pick" | "file";

// ---------------------------------------------------------------------------
// Provider cards for the picker
// ---------------------------------------------------------------------------

// providers array moved inside component to use translations

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function NewSupplierPage() {
  const t = useTranslations("suppliers");
  const tc = useTranslations("common");
  const router = useRouter();
  const [step, setStep] = useState<Step>("pick");

  const providers = [
    {
      key: "btp" as const,
      logo: "/logos/btp.svg",
      name: "BTP.pro",
      description: t("btpDescription"),
    },
    {
      key: "file" as const,
      icon: FileText,
      name: t("fileUrl"),
      description: t("fileUrlDescription"),
    },
  ];

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
          toast.success(t("dostawcazostałutworzony"));
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

      toast.success(t("dostawcazostałutworzonytrybhybrydowyxmlapi"));
      router.push("/suppliers");
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setFileSubmitting(false);
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
    pick: t("newSupplier"),
    file: t("newSupplierFileUrl"),
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
                onClick={() => {
                  if (p.key === "btp") {
                    router.push("/suppliers/new/btp");
                  } else {
                    setStep("file");
                  }
                }}
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
              <CardTitle>{t("supplierData")}</CardTitle>
              <CardDescription>
                {t("dodajDostawceImportujacegoProduktyZPlikuLub")}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form
                onSubmit={handleSubmit(onFileSubmit)}
                className="space-y-4"
              >
                <div className="space-y-2">
                  <Label htmlFor="name">{tc("name")} *</Label>
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
                  <Label htmlFor="code">{t("supplierCode")}</Label>
                  <Input
                    id="code"
                    {...register("code")}
                    placeholder="np. ABC123"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="feed_url">{t("urlFeedaProduktow")}</Label>
                  <Input
                    id="feed_url"
                    {...register("feed_url")}
                    placeholder="https://example.com/feed.xml"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>{t("feedFormat")}</Label>
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
                      {t("interwałSynchronizacji")}
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
                        <SelectItem value="15">{t("every15Min")}</SelectItem>
                        <SelectItem value="30">{t("every30Min")}</SelectItem>
                        <SelectItem value="60">{t("co1Godzine")}</SelectItem>
                        <SelectItem value="120">{t("every2Hours")}</SelectItem>
                        <SelectItem value="360">{t("every6Hours")}</SelectItem>
                        <SelectItem value="720">{t("every12Hours")}</SelectItem>
                        <SelectItem value="1440">{t("onceDaily")}</SelectItem>
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
                        {t("btpApiKeysOptional")}
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
                          {t("podajKluczeApiZPaneluBtpAby")}
                          {t("hybrydowyPełnyKatalogZXmlAktualizacjeStanow")}
                          {t("przezApiMiedzySynchronizacjami")}
                        </p>
                        <div className="space-y-2">
                          <Label htmlFor="xml-public-key">
                            {t("publicKeyLogin")}
                          </Label>
                          <Input
                            id="xml-public-key"
                            value={xmlPublicKey}
                            onChange={(e) => setXmlPublicKey(e.target.value)}
                            placeholder={t("publicKeyFromBtp")}
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="xml-private-key">
                            {t("privateKeyPassword")}
                          </Label>
                          <div className="relative">
                            <Input
                              id="xml-private-key"
                              type={showXmlPrivateKey ? "text" : "password"}
                              className="pr-10"
                              value={xmlPrivateKey}
                              onChange={(e) => setXmlPrivateKey(e.target.value)}
                              placeholder={t("privateKeyFromBtp")}
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
                                  ? t("hideKey")
                                  : t("pokazKlucz")
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
                            {t("apiAddress")}{" "}
                            <span className="text-muted-foreground font-normal">
                              ({t("optional")})
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
                        {tc("creating")}
                      </>
                    ) : (
                      t("utworzDostawce")
                    )}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setStep("pick")}
                  >
                    {tc("back")}
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
