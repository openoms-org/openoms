import Link from "next/link";
import type * as React from "react";
import type { LucideIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface EmptyStateAction {
  label: string;
  href: string;
  variant?: React.ComponentProps<typeof Button>["variant"];
}

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: EmptyStateAction;
  secondaryAction?: EmptyStateAction;
  variant?: "default" | "compact";
  className?: string;
}

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  secondaryAction,
  variant = "default",
  className,
}: EmptyStateProps) {
  const isCompact = variant === "compact";

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center",
        isCompact ? "py-8" : "py-12",
        className
      )}
    >
      <div
        data-testid="empty-state-icon"
        aria-hidden="true"
        className={cn(
          "mb-4 flex items-center justify-center rounded-full border bg-muted/50 text-muted-foreground",
          isCompact ? "h-12 w-12" : "h-16 w-16"
        )}
      >
        <Icon className={isCompact ? "h-5 w-5" : "h-7 w-7"} />
      </div>
      <h3 className={cn("font-semibold text-foreground", isCompact ? "text-base" : "text-lg")}>
        {title}
      </h3>
      <p className="mt-1 max-w-sm text-sm leading-6 text-muted-foreground">
        {description}
      </p>
      {(action || secondaryAction) && (
        <div className="mt-5 flex w-full max-w-sm flex-col gap-2 sm:w-auto sm:max-w-none sm:flex-row sm:justify-center">
          {action && (
            <Button asChild variant={action.variant} className="w-full sm:w-auto">
              <Link href={action.href}>{action.label}</Link>
            </Button>
          )}
          {secondaryAction && (
            <Button asChild variant={secondaryAction.variant ?? "outline"} className="w-full sm:w-auto">
              <Link href={secondaryAction.href}>{secondaryAction.label}</Link>
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
