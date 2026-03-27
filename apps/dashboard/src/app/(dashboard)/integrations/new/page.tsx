"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { ExternalLink } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useCreateIntegration } from "@/hooks/use-integrations";
import { getErrorMessage } from "@/lib/api-client";
import { PageHeader } from "@/components/shared/page-header";
import { IntegrationForm } from "@/components/integrations/integration-form";
import {
  PROVIDERS_WITH_DEDICATED_PAGES,
  INTEGRATION_PROVIDER_LABELS,
} from "@/lib/constants";
import { Button } from "@/components/ui/button";
import { DevelopmentBadge } from "@/components/shared/development-banner";
import { isInDevelopment } from "@/lib/integration-status";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import type { CreateIntegrationRequest } from "@/types/api";
import { useTranslations } from "next-intl";

export default function NewIntegrationPage() {
  const t = useTranslations("integrations");
  const router = useRouter();
  const createIntegration = useCreateIntegration();

  const handleSubmit = (data: CreateIntegrationRequest) => {
    createIntegration.mutate(data, {
      onSuccess: () => {
        toast.success(t("integrationCreated"));
        router.push("/integrations");
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const dedicatedEntries = Object.entries(PROVIDERS_WITH_DEDICATED_PAGES);

  return (
    <AdminGuard>
      <PageHeader
        title={t("newIntegration")}
        description={t("addNewExternalConnection")}
      />

      {dedicatedEntries.length > 0 && (
        <Card className="max-w-2xl mb-6">
          <CardHeader>
            <CardTitle className="text-base">{t("integrationsWithDedicatedWizard")}</CardTitle>
            <CardDescription>
              {t("integrationsRequireOauthAuth")}
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-3">
            {dedicatedEntries.map(([provider, href]) => (
              <Button key={provider} variant="outline" asChild>
                <Link href={href}>
                  <ExternalLink className="mr-2 h-4 w-4" />
                  {INTEGRATION_PROVIDER_LABELS[provider] ?? provider}
                  {isInDevelopment(provider) && <DevelopmentBadge />}
                </Link>
              </Button>
            ))}
          </CardContent>
        </Card>
      )}

      <Card className="max-w-2xl">
        <CardContent className="pt-6">
          <IntegrationForm
            onSubmit={handleSubmit}
            isLoading={createIntegration.isPending}
          />
        </CardContent>
      </Card>
    </AdminGuard>
  );
}
