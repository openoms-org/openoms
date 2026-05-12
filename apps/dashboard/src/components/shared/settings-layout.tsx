import * as React from "react";
import { Surface } from "@/components/shared/surface";
import { cn } from "@/lib/utils";

export function SettingsLayout({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("grid gap-6 lg:grid-cols-[260px_minmax(0,1fr)]", className)}
      {...props}
    />
  );
}

export function SettingsNav({
  className,
  ...props
}: React.ComponentProps<"nav">) {
  return (
    <nav
      className={cn("space-y-1 lg:sticky lg:top-20 lg:self-start", className)}
      {...props}
    />
  );
}

interface SettingsPanelProps extends React.ComponentProps<"section"> {
  title: string;
  description?: string;
  actions?: React.ReactNode;
}

export function SettingsPanel({
  title,
  description,
  actions,
  className,
  children,
  ...props
}: SettingsPanelProps) {
  return (
    <Surface className={cn("space-y-5", className)} {...props}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-base font-semibold text-foreground">{title}</h2>
          {description && (
            <p className="mt-1 text-sm leading-6 text-muted-foreground">{description}</p>
          )}
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
      <div>{children}</div>
    </Surface>
  );
}
