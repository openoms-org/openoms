import * as React from "react";
import { ActionBar } from "@/components/shared/action-bar";
import { Surface } from "@/components/shared/surface";
import { cn } from "@/lib/utils";

interface FormSectionProps extends React.ComponentProps<"section"> {
  title: string;
  description?: string;
}

export function FormSection({
  title,
  description,
  className,
  children,
  ...props
}: FormSectionProps) {
  return (
    <Surface className={cn("space-y-5", className)} {...props}>
      <div>
        <h2 className="text-base font-semibold text-foreground">{title}</h2>
        {description && (
          <p className="mt-1 text-sm leading-6 text-muted-foreground">{description}</p>
        )}
      </div>
      <div className="grid gap-4">{children}</div>
    </Surface>
  );
}

type FormActionsProps = React.ComponentProps<typeof ActionBar>;

export function FormActions({
  className,
  primary,
  secondary,
  meta,
  sticky,
  ...props
}: FormActionsProps) {
  return (
    <ActionBar
      className={cn("mt-2", className)}
      primary={primary}
      secondary={secondary}
      meta={meta}
      sticky={sticky}
      {...props}
    />
  );
}
