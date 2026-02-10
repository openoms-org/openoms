"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { ORDER_STATUSES, ORDER_SOURCES, PAYMENT_STATUSES } from "@/lib/constants";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";

interface OrderFilters {
  status?: string;
  source?: string;
  search?: string;
  payment_status?: string;
  tag?: string;
}

interface OrderFiltersProps {
  filters: OrderFilters;
  onFilterChange: (filters: OrderFilters) => void;
}

const SOURCE_LABELS: Record<string, string> = {
  manual: "Ręczne",
  allegro: "Allegro",
  woocommerce: "WooCommerce",
};

export function OrderFilters({ filters, onFilterChange }: OrderFiltersProps) {
  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;

  return (
    <div className="flex flex-wrap items-center gap-4">
      <Input
        placeholder="Szukaj klienta (imię, email, telefon)..."
        value={filters.search || ""}
        onChange={(e) =>
          onFilterChange({ ...filters, search: e.target.value || undefined })
        }
        className="w-[300px]"
      />
      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">Status:</span>
        <Select
          value={filters.status || "all"}
          onValueChange={(value) =>
            onFilterChange({ ...filters, status: value === "all" ? undefined : value })
          }
        >
          <SelectTrigger className="w-[180px]" size="sm">
            <SelectValue placeholder="Wszystkie" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie</SelectItem>
            {Object.entries(orderStatuses).map(([key, config]) => (
              <SelectItem key={key} value={key}>
                {config.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">Źródło:</span>
        <Select
          value={filters.source || "all"}
          onValueChange={(value) =>
            onFilterChange({ ...filters, source: value === "all" ? undefined : value })
          }
        >
          <SelectTrigger className="w-[180px]" size="sm">
            <SelectValue placeholder="Wszystkie" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie</SelectItem>
            {ORDER_SOURCES.map((source) => (
              <SelectItem key={source} value={source}>
                {SOURCE_LABELS[source] || source}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">Płatność:</span>
        <Select
          value={filters.payment_status || "all"}
          onValueChange={(v) =>
            onFilterChange({ ...filters, payment_status: v === "all" ? undefined : v })
          }
        >
          <SelectTrigger className="w-[180px]" size="sm">
            <SelectValue placeholder="Płatność" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie płatności</SelectItem>
            {Object.entries(PAYMENT_STATUSES).map(([key, { label }]) => (
              <SelectItem key={key} value={key}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <Input
        placeholder="Filtruj po tagu..."
        value={filters.tag || ""}
        onChange={(e) =>
          onFilterChange({ ...filters, tag: e.target.value || undefined })
        }
        className="w-[180px]"
      />
    </div>
  );
}
