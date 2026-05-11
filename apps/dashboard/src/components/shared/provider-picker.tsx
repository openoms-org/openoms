"use client";

import type { ProviderInfo } from "@/lib/provider-info";
import { getVisibleProvidersByCategory } from "@/lib/readiness";
import { ProviderCard } from "@/components/shared/provider-card";

interface ProviderPickerProps {
  category: ProviderInfo["category"];
  selected?: string;
  onSelect: (providerKey: string) => void;
}

export function ProviderPicker({ category, selected, onSelect }: ProviderPickerProps) {
  const providers = getVisibleProvidersByCategory(category);

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {providers.map((provider) => (
        <ProviderCard
          key={provider.key}
          provider={provider}
          selected={selected === provider.key}
          onClick={() => onSelect(provider.key)}
        />
      ))}
    </div>
  );
}
