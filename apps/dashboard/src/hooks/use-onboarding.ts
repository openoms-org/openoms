"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { useAuthStore } from "@/lib/auth";

interface OnboardingSettings {
  dismissed: boolean;
  completed_at?: string;
}

interface OnboardingStep {
  key: string;
  title: string;
  description: string;
  href: string;
  completed: boolean;
}

interface Integration {
  provider: string;
  status: string;
}

interface DashboardStats {
  order_counts?: { total?: number };
  product_count?: number;
}

export function useOnboarding() {
  const queryClient = useQueryClient();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  const { data: settings } = useQuery({
    queryKey: ["settings", "onboarding"],
    queryFn: () => apiClient<OnboardingSettings>("/v1/settings/onboarding"),
    enabled: isAuthenticated,
  });

  const { data: company } = useQuery({
    queryKey: ["settings", "company"],
    queryFn: () => apiClient<{ name?: string; nip?: string }>("/v1/settings/company"),
    enabled: isAuthenticated,
  });

  const { data: integrations } = useQuery({
    queryKey: ["integrations"],
    queryFn: () => apiClient<Integration[]>("/v1/integrations"),
    enabled: isAuthenticated,
  });

  const { data: stats } = useQuery({
    queryKey: ["dashboard-stats"],
    queryFn: () => apiClient<DashboardStats>("/v1/stats/dashboard"),
    enabled: isAuthenticated,
  });

  const dismiss = useMutation({
    mutationFn: () =>
      apiClient("/v1/settings/onboarding", {
        method: "PUT",
        body: JSON.stringify({ dismissed: true }),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["settings", "onboarding"] }),
  });

  const steps: OnboardingStep[] = [
    {
      key: "company",
      title: "Uzupe\u0142nij dane firmy",
      description: "Dodaj nazw\u0119 firmy i NIP \u2014 potrzebne do faktur i przesy\u0142ek.",
      href: "/settings/company",
      completed: !!(company?.name && company?.nip),
    },
    {
      key: "integration",
      title: "Po\u0142\u0105cz z Allegro",
      description: "Zaimportuj zam\u00f3wienia i produkty z Allegro.",
      href: "/integrations",
      completed: Array.isArray(integrations) && integrations.some((i) => i.provider === "allegro" && i.status === "active"),
    },
    {
      key: "product",
      title: "Dodaj pierwszy produkt",
      description: "R\u0119cznie lub przez import CSV.",
      href: "/products",
      completed: (stats?.order_counts?.total ?? 0) > 0 || (stats?.product_count ?? 0) > 0,
    },
    {
      key: "order",
      title: "Pierwsze zam\u00f3wienie",
      description: "Przyjdzie automatycznie z Allegro lub dodaj r\u0119cznie.",
      href: "/orders",
      completed: (stats?.order_counts?.total ?? 0) > 0,
    },
  ];

  const completedCount = steps.filter((s) => s.completed).length;
  const allCompleted = completedCount === steps.length;
  const isVisible = settings !== undefined && !settings.dismissed && !allCompleted;

  return { steps, completedCount, allCompleted, isVisible, dismiss: dismiss.mutate };
}
