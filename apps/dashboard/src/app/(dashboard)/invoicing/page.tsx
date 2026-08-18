"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { KeyRound, Receipt, Trash2, Loader2, Save } from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import { PageHeader } from "@/components/shared/page-header";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { ProviderLogo } from "@/components/shared/provider-logo";
import { StatusBadge } from "@/components/shared/status-badge";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  useIntegrationsByCategory,
  useDeleteIntegration,
} from "@/hooks/use-integrations";
import {
  useInvoicingSettings,
  useUpdateInvoicingSettings,
} from "@/hooks/use-invoices";
import {
  INTEGRATION_STATUSES,
  INVOICING_PROVIDER_LABELS,
  ORDER_STATUSES,
} from "@/lib/constants";
import { formatDate } from "@/lib/utils";
import { getErrorMessage } from "@/lib/api-client";
import type { InvoicingSettings } from "@/types/api";
import { useEffectSyncedState } from "@/hooks/use-effect-synced-state";

const DEFAULT_SETTINGS: InvoicingSettings = {
  provider: "",
  auto_create_on_status: [],
  default_tax_rate: 23,
  payment_days: 14,
  credentials: {},
};

export default function InvoicingPage() {
  const router = useRouter();
  const t = useTranslations("invoices");

  // ── Integrations ──
  const { invoicing, isLoading, isError, refetch } =
    useIntegrationsByCategory();
  const deleteIntegration = useDeleteIntegration();
  const [deleteId, setDeleteId] = useState<string | null>(null);

  // ── Settings ──
  const { data: settings, isLoading: settingsLoading } =
    useInvoicingSettings();
  const updateSettings = useUpdateInvoicingSettings();
  const settingsForm = useMemo(
    () => ({
      ...DEFAULT_SETTINGS,
      ...settings,
      credentials: settings?.credentials || {},
    }),
    [settings],
  );
  const [form, setForm] = useEffectSyncedState(
    settingsForm,
    JSON.stringify(settingsForm),
  );

  const handleSave = async () => {
    try {
      await updateSettings.mutateAsync(form);
      toast.success(t("invoicingSettingsSaved"));
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("invoicingSettingsSaveFailed");
      toast.error(message);
    }
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteIntegration.mutate(deleteId, {
      onSuccess: () => {
        toast.success(t("invoicingProviderDeleted"));
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  if (isLoading || settingsLoading) {
    return <LoadingSkeleton />;
  }

  return (
    <AdminGuard>
      <PageHeader
        title="Fakturowanie"
        description="Zarzadzaj dostawcami faktur i ustawieniami fakturowania"
      />

      {/* ── Invoicing Settings ── */}
      <div className="space-y-6 mb-10">
        {/* Provider selection */}
        <Card>
          <CardHeader>
            <CardTitle>Dostawca faktur</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Dostawca</Label>
              <Select
                value={form.provider || "none"}
                onValueChange={(value) =>
                  setForm({ ...form, provider: value === "none" ? "" : value })
                }
              >
                <SelectTrigger className="w-full max-w-xs">
                  <SelectValue placeholder="Wybierz dostawce" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Brak</SelectItem>
                  {Object.entries(INVOICING_PROVIDER_LABELS).map(
                    ([key, label]) => (
                      <SelectItem key={key} value={key}>
                        {label}
                      </SelectItem>
                    )
                  )}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* Credentials — Fakturownia */}
        {form.provider === "fakturownia" && (
          <Card>
            <CardHeader>
              <CardTitle>Dane dostepowe Fakturownia</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Uzyskaj dane z{" "}
                <a
                  href="https://fakturownia.pl"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="underline"
                >
                  fakturownia.pl
                </a>
              </p>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label>Subdomena</Label>
                  <Input
                    value={form.credentials.subdomain || ""}
                    onChange={(e) =>
                      setForm({
                        ...form,
                        credentials: {
                          ...form.credentials,
                          subdomain: e.target.value,
                        },
                      })
                    }
                    placeholder="moja-firma"
                  />
                  <p className="text-xs text-muted-foreground">
                    np. moja-firma dla moja-firma.fakturownia.pl
                  </p>
                </div>
                <div className="space-y-2">
                  <Label>Token API</Label>
                  <Input
                    type="password"
                    value={form.credentials.api_token || ""}
                    onChange={(e) =>
                      setForm({
                        ...form,
                        credentials: {
                          ...form.credentials,
                          api_token: e.target.value,
                        },
                      })
                    }
                    placeholder="Token API z Fakturowni"
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Credentials — wFirma */}
        {form.provider === "wfirma" && (
          <Card>
            <CardHeader>
              <CardTitle>Dane dostepowe wFirma</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Wygeneruj klucz API w panelu{" "}
                <a
                  href="https://wfirma.pl"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="underline"
                >
                  wfirma.pl
                </a>{" "}
                w sekcji Ustawienia &rarr; Integracje &rarr; API.
              </p>
              <div className="max-w-sm space-y-2">
                <Label>Klucz API</Label>
                <Input
                  type="password"
                  value={form.credentials.api_key || ""}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      credentials: {
                        ...form.credentials,
                        api_key: e.target.value,
                      },
                    })
                  }
                  placeholder="Klucz API z wFirma"
                />
              </div>
            </CardContent>
          </Card>
        )}

        {/* Credentials — inFakt */}
        {form.provider === "infakt" && (
          <Card>
            <CardHeader>
              <CardTitle>Dane dostepowe inFakt</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Wygeneruj klucz API w panelu{" "}
                <a
                  href="https://app.infakt.pl"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="underline"
                >
                  app.infakt.pl
                </a>{" "}
                w sekcji Ustawienia &rarr; Integracje.
              </p>
              <div className="max-w-sm space-y-2">
                <Label>Klucz API</Label>
                <Input
                  type="password"
                  value={form.credentials.api_key || ""}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      credentials: {
                        ...form.credentials,
                        api_key: e.target.value,
                      },
                    })
                  }
                  placeholder="Klucz API z inFakt"
                />
              </div>
            </CardContent>
          </Card>
        )}

        {/* Invoice settings */}
        <Card>
          <CardHeader>
            <CardTitle>Ustawienia faktur</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Domyslna stawka VAT (%)</Label>
                <Input
                  type="number"
                  value={form.default_tax_rate}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      default_tax_rate: parseInt(e.target.value) || 23,
                    })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>Termin platnosci (dni)</Label>
                <Input
                  type="number"
                  value={form.payment_days}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      payment_days: parseInt(e.target.value) || 14,
                    })
                  }
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Auto-create triggers */}
        <Card>
          <CardHeader>
            <CardTitle>Automatyczne wystawianie</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground mb-4">
              Wybierz przy jakich zmianach statusu zamowienia automatycznie
              wystawiac fakture
            </p>
            <div className="space-y-3">
              {Object.entries(ORDER_STATUSES).map(([key, { label }]) => (
                <div key={key} className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    id={`auto-invoice-${key}`}
                    checked={form.auto_create_on_status.includes(key)}
                    onChange={(e) => {
                      setForm({
                        ...form,
                        auto_create_on_status: e.target.checked
                          ? [...form.auto_create_on_status, key]
                          : form.auto_create_on_status.filter(
                              (s) => s !== key
                            ),
                      });
                    }}
                    className="h-4 w-4 rounded border-gray-300"
                  />
                  <label
                    htmlFor={`auto-invoice-${key}`}
                    className="text-sm"
                  >
                    {label}
                  </label>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Save button */}
        <div className="flex justify-end">
          <Button onClick={handleSave} disabled={updateSettings.isPending}>
            {updateSettings.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            Zapisz ustawienia
          </Button>
        </div>
      </div>

      {/* ── Connected Integrations ── */}
      <div>
        <h2 className="text-lg font-semibold mb-4">
          Polaczone integracje fakturowe
        </h2>

        {isError && (
          <div className="rounded-md border border-destructive bg-destructive/10 p-4 mb-4">
            <p className="text-sm text-destructive">
              Wystapil blad podczas ladowania integracji. Sprobuj odswiezyc
              strone.
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-2"
              onClick={() => refetch()}
            >
              Sprobuj ponownie
            </Button>
          </div>
        )}

        {!invoicing || invoicing.length === 0 ? (
          <EmptyState
            icon={Receipt}
            title="Brak polaczonych dostawcow faktur"
            description="Dostawcy faktur pojawia sie tutaj po dodaniu integracji fakturowej."
          />
        ) : (
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Dostawca</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Dane uwierzytelniajace</TableHead>
                  <TableHead>Ostatnia synchronizacja</TableHead>
                  <TableHead>Utworzono</TableHead>
                  <TableHead className="w-[60px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {invoicing.map((integration) => (
                  <TableRow
                    key={integration.id}
                    className="cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() =>
                      router.push(`/integrations/${integration.id}`)
                    }
                  >
                    <TableCell className="font-medium">
                      <ProviderLogo
                        providerKey={integration.provider}
                        category="invoicing"
                        size="sm"
                      />
                    </TableCell>
                    <TableCell>
                      <StatusBadge
                        status={integration.status}
                        statusMap={INTEGRATION_STATUSES}
                        translationPrefix="integration"
                      />
                    </TableCell>
                    <TableCell>
                      {integration.has_credentials ? (
                        <Badge variant="success" className="gap-1">
                          <KeyRound className="h-3 w-3" />
                          Skonfigurowane
                        </Badge>
                      ) : (
                        <Badge
                          variant="secondary"
                          className="text-muted-foreground"
                        >
                          Brak
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      {integration.last_sync_at
                        ? formatDate(integration.last_sync_at)
                        : "---"}
                    </TableCell>
                    <TableCell>{formatDate(integration.created_at)}</TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteId(integration.id);
                        }}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Usun dostawce faktur"
        description="Czy na pewno chcesz usunac tego dostawce faktur? Ta operacja jest nieodwracalna."
        confirmLabel="Usun"
        variant="destructive"
        onConfirm={handleDelete}
        isPending={deleteIntegration.isPending}
      />
    </AdminGuard>
  );
}
