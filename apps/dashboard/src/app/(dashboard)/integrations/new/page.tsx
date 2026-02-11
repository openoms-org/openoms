"use client";

import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useCreateIntegration } from "@/hooks/use-integrations";
import { getErrorMessage } from "@/lib/api-client";
import { PageHeader } from "@/components/shared/page-header";
import { IntegrationForm } from "@/components/integrations/integration-form";
import { Card, CardContent } from "@/components/ui/card";
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

  return (
    <AdminGuard>
      <PageHeader
        title="Nowa integracja"
        description="Dodaj nowe połączenie z zewnętrznym serwisem"
      />
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
