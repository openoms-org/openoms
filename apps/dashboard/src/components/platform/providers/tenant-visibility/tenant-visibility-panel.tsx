"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Info, UserPlus } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ActionDialog } from "@/components/shared/action-dialog";
import { PublicationStateBadge } from "@/components/platform/publication-state-badge";
import { ApiClientError } from "@/lib/api-client";
import { useEnableTenantForVersion } from "@/hooks/use-platform-provider-validation";
import type { ProviderVersion } from "@/types/platform";

interface TenantVisibilityPanelProps {
  providerId: string;
  version: ProviderVersion;
}

const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/**
 * Screen 10 — Tenant Visibility. Shows public availability and lets a platform
 * admin add a tenant to a version's private-beta allowlist (the one backend
 * endpoint that exists: POST .../enable-tenant). Listing the current allowlist,
 * per-tenant validation/health and tenant capability downgrades have no backend
 * yet, so those sections render an explicit "not yet available" state.
 */
export function TenantVisibilityPanel({
  providerId,
  version,
}: TenantVisibilityPanelProps) {
  const t = useTranslations("platform");
  const enableTenant = useEnableTenantForVersion(providerId, version.id);

  const [open, setOpen] = useState(false);
  const [tenantId, setTenantId] = useState("");

  const isPrivateBeta = version.publication_state === "private_beta";
  const valid = UUID_RE.test(tenantId.trim());

  const submit = () => {
    enableTenant.mutate(
      { tenant_id: tenantId.trim() },
      {
        onSuccess: () => {
          toast.success(t("tenantVisibility.enabled"));
          setOpen(false);
          setTenantId("");
        },
        onError: (err) =>
          toast.error(err instanceof ApiClientError ? err.message : t("saveError")),
      },
    );
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t("tenantVisibility.title")}</CardTitle>
          <CardDescription>{t("tenantVisibility.description")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm text-muted-foreground">
              {t("tenantVisibility.publicAvailability")}
            </span>
            <PublicationStateBadge state={version.publication_state} />
            {version.publication_state === "available" && (
              <Badge variant="success">{t("tenantVisibility.allTenants")}</Badge>
            )}
          </div>

          {isPrivateBeta ? (
            <div className="space-y-2">
              <p className="text-sm">{t("tenantVisibility.privateBetaExplainer")}</p>
              <Button type="button" size="sm" onClick={() => setOpen(true)}>
                <UserPlus className="h-4 w-4" />
                {t("tenantVisibility.enableTenant")}
              </Button>
            </div>
          ) : (
            <p className="flex items-start gap-2 rounded-md border bg-muted/40 p-3 text-sm text-muted-foreground">
              <Info className="mt-0.5 h-4 w-4 shrink-0" />
              {t("tenantVisibility.onlyPrivateBeta")}
            </p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("tenantVisibility.allowlistTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="flex items-start gap-2 rounded-md border border-dashed bg-muted/30 p-3 text-sm text-muted-foreground">
            <Info className="mt-0.5 h-4 w-4 shrink-0" />
            {t("tenantVisibility.allowlistUnavailable")}
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("tenantVisibility.downgradesTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="flex items-start gap-2 rounded-md border border-dashed bg-muted/30 p-3 text-sm text-muted-foreground">
            <Info className="mt-0.5 h-4 w-4 shrink-0" />
            {t("tenantVisibility.downgradesUnavailable")}
          </p>
        </CardContent>
      </Card>

      <ActionDialog
        open={open}
        onOpenChange={(o) => {
          if (!o) {
            setOpen(false);
            setTenantId("");
          }
        }}
        title={t("tenantVisibility.enableTenantTitle")}
        description={t("tenantVisibility.enableTenantDescription")}
        confirmLabel={t("tenantVisibility.enableTenantAction")}
        isPending={enableTenant.isPending}
        confirmDisabled={!valid}
        onConfirm={submit}
      >
        <div className="space-y-1.5">
          <label className="text-sm font-medium" htmlFor="tenant-id">
            {t("tenantVisibility.tenantId")}
          </label>
          <Input
            id="tenant-id"
            value={tenantId}
            onChange={(e) => setTenantId(e.target.value)}
            placeholder="00000000-0000-0000-0000-000000000000"
            aria-invalid={tenantId.length > 0 && !valid}
          />
          {tenantId.length > 0 && !valid && (
            <p className="text-xs text-destructive">
              {t("tenantVisibility.tenantIdInvalid")}
            </p>
          )}
        </div>
      </ActionDialog>
    </div>
  );
}
