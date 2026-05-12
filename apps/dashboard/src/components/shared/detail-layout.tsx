import * as React from "react";
import { cn } from "@/lib/utils";

export function DetailLayout({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]", className)}
      {...props}
    />
  );
}

export function DetailMain({
  className,
  role,
  "aria-label": ariaLabel,
  "aria-labelledby": ariaLabelledBy,
  ...props
}: React.ComponentProps<"section">) {
  return (
    <section
      role={role ?? (ariaLabel || ariaLabelledBy ? "region" : undefined)}
      aria-label={ariaLabel}
      aria-labelledby={ariaLabelledBy}
      className={cn("min-w-0 space-y-4", className)}
      {...props}
    />
  );
}

export function DetailSidebar({
  className,
  "aria-label": ariaLabel,
  "aria-labelledby": ariaLabelledBy,
  ...props
}: React.ComponentProps<"aside">) {
  return (
    <aside
      aria-label={ariaLabel}
      aria-labelledby={ariaLabelledBy}
      className={cn("min-w-0 space-y-4", className)}
      {...props}
    />
  );
}
