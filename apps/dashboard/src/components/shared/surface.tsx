import * as React from "react";
import { cn } from "@/lib/utils";

type SurfaceTone = "default" | "muted" | "success" | "warning" | "danger";

const toneClasses: Record<SurfaceTone, string> = {
  default: "border-border bg-card text-card-foreground",
  muted: "border-border bg-muted/35 text-foreground",
  success: "border-emerald-200 bg-emerald-50 text-emerald-950",
  warning: "border-amber-200 bg-amber-50 text-amber-950",
  danger: "border-red-200 bg-red-50 text-red-950",
};

interface SurfaceProps extends React.ComponentProps<"section"> {
  tone?: SurfaceTone;
  padded?: boolean;
}

export function Surface({
  tone = "default",
  padded = true,
  className,
  role,
  "aria-label": ariaLabel,
  "aria-labelledby": ariaLabelledBy,
  ...props
}: SurfaceProps) {
  return (
    <section
      role={role ?? (ariaLabel || ariaLabelledBy ? "region" : undefined)}
      aria-label={ariaLabel}
      aria-labelledby={ariaLabelledBy}
      className={cn(
        "rounded-lg border",
        toneClasses[tone],
        padded && "p-4 sm:p-5",
        className
      )}
      {...props}
    />
  );
}
