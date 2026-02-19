"use client";

import { FileText, Plug } from "lucide-react";
import { cn } from "@/lib/utils";

type IntegrationMethod = "file" | "api";

interface IntegrationMethodSelectorProps {
  selected: IntegrationMethod | null;
  onSelect: (method: IntegrationMethod) => void;
}

const methods = [
  {
    value: "file" as const,
    icon: FileText,
    title: "Plik / URL",
    description: "Import produktow z pliku IOF, CSV lub URL feeda",
  },
  {
    value: "api" as const,
    icon: Plug,
    title: "API (BTP.pro)",
    description: "Polaczenie z hurtownia BTP przez API",
  },
];

export function IntegrationMethodSelector({
  selected,
  onSelect,
}: IntegrationMethodSelectorProps) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      {methods.map((method) => {
        const Icon = method.icon;
        return (
          <button
            key={method.value}
            type="button"
            onClick={() => onSelect(method.value)}
            className={cn(
              "flex flex-col items-start gap-3 rounded-lg border p-6 text-left transition-colors hover:bg-muted/50",
              selected === method.value && "border-primary bg-primary/5",
            )}
          >
            <Icon className="h-8 w-8 text-muted-foreground" />
            <div>
              <p className="font-medium text-base">{method.title}</p>
              <p className="text-sm text-muted-foreground">
                {method.description}
              </p>
            </div>
          </button>
        );
      })}
    </div>
  );
}
