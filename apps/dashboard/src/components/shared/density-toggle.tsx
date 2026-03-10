"use client";

import { AlignJustify, AlignCenter, AlignLeft } from "lucide-react";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import { useTableDensity, type TableDensity } from "@/lib/table-density";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

const options: { value: TableDensity; icon: typeof AlignJustify; labelKey: string }[] = [
  { value: "compact", icon: AlignJustify, labelKey: "compact" },
  { value: "comfortable", icon: AlignCenter, labelKey: "comfortable" },
  { value: "spacious", icon: AlignLeft, labelKey: "spacious" },
];

export function DensityToggle() {
  const { density, setDensity } = useTableDensity();
  const t = useTranslations("shared.tableDensity");

  return (
    <TooltipProvider delayDuration={0}>
      <div className="flex items-center rounded-md border bg-muted/50 p-0.5">
        {options.map(({ value, icon: Icon, labelKey }) => (
          <Tooltip key={value}>
            <TooltipTrigger asChild>
              <button
                onClick={() => setDensity(value)}
                className={cn(
                  "rounded-sm p-1.5 transition-colors",
                  density === value
                    ? "bg-background text-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                <Icon className="h-3.5 w-3.5" />
              </button>
            </TooltipTrigger>
            <TooltipContent side="bottom">
              <p className="text-xs">{t(labelKey)}</p>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>
    </TooltipProvider>
  );
}
