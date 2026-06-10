"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import type { ProviderPublicationState } from "@/types/platform";

type BadgeVariant = React.ComponentProps<typeof Badge>["variant"];

const STATE_VARIANTS: Record<ProviderPublicationState, BadgeVariant> = {
  research: "secondary",
  designed: "secondary",
  adapter_in_progress: "info",
  internal_validation: "info",
  private_beta: "warning",
  available: "success",
  deprecated: "warning",
  retired: "outline",
};

interface PublicationStateBadgeProps {
  state: ProviderPublicationState | null | undefined;
}

/**
 * Renders a provider version publication_state as a tone-coded badge with a
 * localized label. Falls back to a neutral dash when no state is known.
 */
export function PublicationStateBadge({ state }: PublicationStateBadgeProps) {
  const t = useTranslations("platform");

  if (!state) {
    return <span className="text-muted-foreground">—</span>;
  }

  return (
    <Badge variant={STATE_VARIANTS[state] ?? "outline"}>
      {t(`publicationState.${state}`)}
    </Badge>
  );
}
