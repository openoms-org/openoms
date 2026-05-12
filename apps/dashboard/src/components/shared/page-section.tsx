import * as React from "react";
import { Surface } from "@/components/shared/surface";
import { cn } from "@/lib/utils";

interface PageSectionProps extends React.ComponentProps<"section"> {
  title?: string;
  description?: string;
  actions?: React.ReactNode;
  framed?: boolean;
  contentClassName?: string;
}

export function PageSection({
  title,
  description,
  actions,
  framed = true,
  className,
  contentClassName,
  children,
  ...props
}: PageSectionProps) {
  const hasHeader = title || description || actions;
  const content = (
    <>
      {hasHeader && (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            {title && (
              <h2 className="text-base font-semibold tracking-normal text-foreground">
                {title}
              </h2>
            )}
            {description && (
              <p className="mt-1 text-sm leading-6 text-muted-foreground">
                {description}
              </p>
            )}
          </div>
          {actions && (
            <div className="flex shrink-0 items-center gap-2 sm:justify-end">{actions}</div>
          )}
        </div>
      )}
      <div className={contentClassName}>{children}</div>
    </>
  );

  if (framed) {
    return (
      <Surface className={cn("space-y-4", className)} {...props}>
        {content}
      </Surface>
    );
  }

  return (
    <section className={cn("space-y-4", className)} {...props}>
      {content}
    </section>
  );
}
