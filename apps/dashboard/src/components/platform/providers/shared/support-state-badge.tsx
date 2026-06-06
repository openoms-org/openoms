"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { supportStatusTone } from "@/components/platform/providers/capability-matrix/capability-helpers";
import type { SupportStatus } from "@/types/platform";

interface SupportStateBadgeProps {
  status: SupportStatus;
}

/**
 * Tone-coded support-state badge. Meaning is never color-only: the badge always
 * shows a translated label and a tooltip describing what the state means.
 */
export function SupportStateBadge({ status }: SupportStateBadgeProps) {
  const t = useTranslations("platform");

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge variant={supportStatusTone(status)}>
            {t(`capabilityMatrix.supportStatus.${status}`)}
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          {t(`capabilityMatrix.supportStatusHint.${status}`)}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
