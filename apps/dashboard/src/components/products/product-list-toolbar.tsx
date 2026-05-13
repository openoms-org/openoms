import * as React from "react";

import { cn } from "@/lib/utils";

interface ProductListToolbarProps extends React.ComponentProps<"div"> {
  filters: React.ReactNode;
  actions?: React.ReactNode;
}

export function ProductListToolbar({
  filters,
  actions,
  className,
  ...props
}: ProductListToolbarProps) {
  return (
    <div className={cn("space-y-3", className)} {...props}>
      <div
        data-testid="product-list-toolbar-filters"
        className="flex min-w-0 flex-wrap items-center gap-3"
      >
        {filters}
      </div>
      {actions && (
        <div
          data-testid="product-list-toolbar-actions"
          className="flex w-full flex-wrap items-center gap-2 sm:justify-end"
        >
          {actions}
        </div>
      )}
    </div>
  );
}
