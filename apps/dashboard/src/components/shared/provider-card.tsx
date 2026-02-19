"use client";

import { cn } from "@/lib/utils";
import type { ProviderInfo } from "@/lib/provider-info";
import { Badge } from "@/components/ui/badge";

interface ProviderCardProps {
  provider: ProviderInfo;
  selected?: boolean;
  onClick?: () => void;
}

export function ProviderCard({ provider, selected, onClick }: ProviderCardProps) {
  const Icon = provider.icon;
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex flex-col items-start gap-2 rounded-lg border p-4 text-left transition-colors hover:bg-muted/50",
        selected && "border-primary bg-primary/5",
      )}
    >
      <div className="flex w-full items-center justify-between">
        <Icon className="h-6 w-6 text-muted-foreground" />
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
