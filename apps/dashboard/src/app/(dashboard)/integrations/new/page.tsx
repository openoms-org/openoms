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
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import type { CreateIntegrationRequest } from "@/types/api";

export default function NewIntegrationPage() {
  const router = useRouter();
  const createIntegration = useCreateIntegration();

  const handleSubmit = (data: CreateIntegrationRequest) => {
    createIntegration.mutate(data, {
      onSuccess: () => {
        toast.success("Integracja została utworzona");
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
        title="Nowa integracja"
        description="Dodaj nowe połączenie z zewnętrznym serwisem"
      />

      {dedicatedEntries.length > 0 && (
        <Card className="max-w-2xl mb-6">
          <CardHeader>
            <CardTitle className="text-base">Integracje z dedykowanym kreatorem</CardTitle>
            <CardDescription>
              Poniższe integracje wymagają autoryzacji OAuth i mają własną stronę konfiguracji.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-3">
            {dedicatedEntries.map(([provider, href]) => (
              <Button key={provider} variant="outline" asChild>
                <Link href={href}>
                  <ExternalLink className="mr-2 h-4 w-4" />
                  {INTEGRATION_PROVIDER_LABELS[provider] ?? provider}
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
