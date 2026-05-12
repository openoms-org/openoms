"use client";

import { cn } from "@/lib/utils";
import type { ProviderInfo } from "@/lib/provider-info";
import { Badge } from "@/components/ui/badge";
import { ProviderLogo } from "@/components/shared/provider-logo";

interface ProviderCardProps {
  provider: ProviderInfo;
  selected?: boolean;
  onClick?: () => void;
}

export function ProviderCard({ provider, selected, onClick }: ProviderCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex flex-col items-start gap-3 rounded-lg border p-4 text-left transition-colors hover:bg-muted/50",
        selected && "border-primary bg-primary/5",
      )}
    >
      <div className="flex w-full items-center justify-between">
        <ProviderLogo provider={provider} size="lg" />
        {provider.beta && (
          <Badge variant="secondary" className="text-xs">
            Beta
          </Badge>
        )}
      </div>
      <div>
        <p className="font-medium">{provider.name}</p>
        <p className="text-sm text-muted-foreground">{provider.description}</p>
      </div>
    </button>
  );
}
