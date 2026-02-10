import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface StatusBadgeProps {
  status: string;
  statusMap: Record<string, { label: string; color: string }>;
}

export function StatusBadge({ status, statusMap }: StatusBadgeProps) {
  const config = statusMap[status];
  if (!config) {
    return <Badge variant="outline">{status}</Badge>;
  }

  return (
    <span className={cn("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium", config.color)}>
      {config.label}
    </span>
  );
}
