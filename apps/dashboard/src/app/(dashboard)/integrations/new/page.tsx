"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth } from "@/hooks/use-auth";
import { useCreateIntegration } from "@/hooks/use-integrations";
import { PageHeader } from "@/components/shared/page-header";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { IntegrationForm } from "@/components/integrations/integration-form";
import { Card, CardContent } from "@/components/ui/card";
import type { CreateIntegrationRequest } from "@/types/api";

export default function NewIntegrationPage() {
  const router = useRouter();
  const { isAdmin, isLoading: authLoading } = useAuth();
  const createIntegration = useCreateIntegration();

  useEffect(() => {
    if (!authLoading && !isAdmin) {
      router.push("/");
    }
  }, [authLoading, isAdmin, router]);

  if (authLoading || !isAdmin) {
    return <LoadingSkeleton />;
  }

  const handleSubmit = (data: CreateIntegrationRequest) => {
    createIntegration.mutate(data, {
      onSuccess: () => {
        toast.success("Integracja zostala utworzona");
        router.push("/integrations");
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : "Blad tworzenia integracji");
      },
    });
  };

  return (
    <>
      <PageHeader
        title="Nowa integracja"
        description="Dodaj nowe polaczenie z zewnetrznym serwisem"
      />
      <Card className="max-w-2xl">
        <CardContent className="pt-6">
          <IntegrationForm
            onSubmit={handleSubmit}
            isLoading={createIntegration.isPending}
          />
        </CardContent>
      </Card>
    </>
  );
}
