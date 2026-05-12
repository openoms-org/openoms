import * as React from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface PageHeaderAction {
  label: string;
  href: string;
}

interface PageHeaderProps {
  title: string;
  description?: string;
  action?: PageHeaderAction;
  actions?: React.ReactNode;
  breadcrumbs?: React.ReactNode;
  meta?: React.ReactNode;
  className?: string;
}

export function PageHeader({
  title,
  description,
  action,
  actions,
  breadcrumbs,
  meta,
  className,
}: PageHeaderProps) {
  return (
    <header className={cn("mb-6 space-y-3", className)}>
      {breadcrumbs}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 space-y-2">
          <div className="space-y-1">
            <h1 className="text-2xl font-semibold tracking-normal text-foreground">
              {title}
            </h1>
            {description && (
              <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
                {description}
              </p>
            )}
          </div>
          {meta && <div className="text-sm text-muted-foreground">{meta}</div>}
        </div>
        {(action || actions) && (
          <div className="flex shrink-0 flex-col gap-2 sm:flex-row sm:items-center">
            {actions}
            {action && (
              <Button asChild>
                <Link href={action.href}>{action.label}</Link>
              </Button>
            )}
          </div>
        )}
      </div>
    </header>
  );
}
