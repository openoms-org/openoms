"use client";

import { AlignJustify, AlignCenter, AlignLeft } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTableDensity, type TableDensity } from "@/lib/table-density";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

const options: { value: TableDensity; icon: typeof AlignJustify; label: string }[] = [
  { value: "compact", icon: AlignJustify, label: "Kompaktowy" },
  { value: "comfortable", icon: AlignCenter, label: "Standardowy" },
  { value: "spacious", icon: AlignLeft, label: "Przestronny" },
];

export function DensityToggle() {
  const { density, setDensity } = useTableDensity();

  return (
    <TooltipProvider delayDuration={0}>
      <div className="flex items-center rounded-md border bg-muted/50 p-0.5">
        {options.map(({ value, icon: Icon, label }) => (
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
              <p className="text-xs">{label}</p>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>
    </TooltipProvider>
  );
}
