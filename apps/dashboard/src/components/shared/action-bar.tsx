import * as React from "react";
import { cn } from "@/lib/utils";

interface ActionBarProps extends React.ComponentProps<"div"> {
  primary?: React.ReactNode;
  secondary?: React.ReactNode;
  meta?: React.ReactNode;
  sticky?: boolean;
}

export function ActionBar({
  primary,
  secondary,
  meta,
  sticky = false,
  className,
  ...props
}: ActionBarProps) {
  return (
    <div
      className={cn(
        "flex flex-col gap-3 border-border bg-background/95 py-3 sm:flex-row sm:items-center sm:justify-between",
        sticky && "sticky bottom-0 z-10 border-t backdrop-blur",
        className
      )}
      {...props}
    >
      <div className="flex min-w-0 flex-1 items-center gap-2 text-sm text-muted-foreground">
        {meta}
      </div>
      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-end">
        {secondary}
        {primary}
      </div>
    </div>
  );
}
