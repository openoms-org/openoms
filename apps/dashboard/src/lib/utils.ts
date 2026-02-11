import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { format } from "date-fns";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatDate(date: string | Date): string {
  return format(new Date(date), "dd.MM.yyyy HH:mm");
}

export function formatDateTime(date: string | Date): string {
  return format(new Date(date), "dd.MM.yyyy HH:mm:ss");
}

export function formatCurrency(amount: number | undefined | null, currency = "PLN"): string {
  if (amount == null || isNaN(amount)) return "0,00 zł";
  return new Intl.NumberFormat("pl-PL", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format(amount);
}

export function shortId(uuid: string): string {
  if (!uuid) return "—";
  return uuid.substring(0, 8);
}
