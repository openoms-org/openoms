"use client";

import { ExternalLink } from "lucide-react";
import { useTranslations } from "next-intl";

interface EvidenceLinkProps {
  /** A URL or free-text reference; URLs render as a safe external link. */
  value: string;
}

const URL_RE = /^https?:\/\/\S+$/i;

/**
 * Renders an evidence reference. When it's an http(s) URL it becomes a
 * noopener/noreferrer external link; otherwise it's shown as plain reference
 * text. A missing reference shows a localized "no evidence" marker.
 */
export function EvidenceLink({ value }: EvidenceLinkProps) {
  const t = useTranslations("platform");
  const trimmed = value.trim();

  if (!trimmed) {
    return (
      <span className="text-xs text-muted-foreground">
        {t("capabilityMatrix.noEvidence")}
      </span>
    );
  }

  if (URL_RE.test(trimmed)) {
    return (
      <a
        href={trimmed}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-1 text-xs text-primary underline-offset-4 hover:underline"
      >
        <ExternalLink className="h-3 w-3" />
        {t("capabilityMatrix.openEvidence")}
      </a>
    );
  }

  return <span className="text-xs text-foreground">{trimmed}</span>;
}
