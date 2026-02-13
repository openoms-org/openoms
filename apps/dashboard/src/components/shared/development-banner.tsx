import { AlertTriangle } from "lucide-react";
import { Badge } from "@/components/ui/badge";

/**
 * Full-width amber banner displayed at the top of pages for integrations
 * that have not yet been verified in a production environment.
 */
export function DevelopmentBanner() {
  return (
    <div className="flex items-start gap-3 rounded-md border border-warning/30 bg-warning/10 p-4">
      <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-warning" />
      <div className="text-sm text-warning">
        <strong>W budowie</strong> &mdash; Ta integracja jest w trakcie budowy i
        nie została jeszcze zweryfikowana produkcyjnie. Używaj na własną
        odpowiedzialność.
      </div>
    </div>
  );
}

/**
 * Small inline badge reading "W budowie", intended for use in tables and lists
 * next to integration provider names that are not yet production-verified.
 */
export function DevelopmentBadge() {
  return (
    <Badge variant="warning" className="ml-2 text-[10px] leading-tight">
      W budowie
    </Badge>
  );
}
