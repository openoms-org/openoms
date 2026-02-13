export type StatusVariant = "success" | "warning" | "error" | "info" | "neutral" | "draft";

export interface StatusColor {
  bg: string;
  text: string;
  darkBg: string;
  darkText: string;
}

export const statusColors: Record<StatusVariant, StatusColor> = {
  success: {
    bg: "bg-success/15",
    text: "text-success",
    darkBg: "dark:bg-success/15",
    darkText: "dark:text-success",
  },
  warning: {
    bg: "bg-warning/15",
    text: "text-warning",
    darkBg: "dark:bg-warning/15",
    darkText: "dark:text-warning",
  },
  error: {
    bg: "bg-destructive/15",
    text: "text-destructive",
    darkBg: "dark:bg-destructive/15",
    darkText: "dark:text-destructive",
  },
  info: {
    bg: "bg-info/15",
    text: "text-info",
    darkBg: "dark:bg-info/15",
    darkText: "dark:text-info",
  },
  neutral: {
    bg: "bg-muted",
    text: "text-muted-foreground",
    darkBg: "dark:bg-muted",
    darkText: "dark:text-muted-foreground",
  },
  draft: {
    bg: "bg-secondary",
    text: "text-secondary-foreground",
    darkBg: "dark:bg-secondary",
    darkText: "dark:text-secondary-foreground",
  },
};

export function statusColorClasses(variant: StatusVariant): string {
  const c = statusColors[variant];
  return `${c.bg} ${c.text} ${c.darkBg} ${c.darkText}`;
}

// Map common status strings to variants
export function getStatusVariant(status: string): StatusVariant {
  const lower = status.toLowerCase();
  if (["active", "completed", "confirmed", "paid", "delivered", "sent", "approved", "connected"].includes(lower)) return "success";
  if (["pending", "processing", "in_progress", "in_transit", "partial", "awaiting"].includes(lower)) return "warning";
  if (["error", "failed", "rejected", "cancelled", "canceled", "overdue", "inactive"].includes(lower)) return "error";
  if (["new", "created", "open", "ready"].includes(lower)) return "info";
  if (["draft", "archived"].includes(lower)) return "draft";
  return "neutral";
}
