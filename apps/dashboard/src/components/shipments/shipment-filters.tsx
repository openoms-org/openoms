"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SHIPMENT_STATUSES, SHIPMENT_PROVIDERS } from "@/lib/constants";

interface ShipmentFilters {
  status?: string;
  provider?: string;
}

interface ShipmentFiltersProps {
  filters: ShipmentFilters;
  onFilterChange: (filters: ShipmentFilters) => void;
}

export function ShipmentFilters({
  filters,
  onFilterChange,
}: ShipmentFiltersProps) {
  return (
    <div className="flex flex-wrap gap-3">
      <Select
        value={filters.status ?? "all"}
        onValueChange={(value) =>
          onFilterChange({
            ...filters,
            status: value === "all" ? undefined : value,
          })
        }
      >
        <SelectTrigger className="w-[180px]">
          <SelectValue placeholder="Status" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Wszystkie statusy</SelectItem>
          {Object.entries(SHIPMENT_STATUSES).map(([key, { label }]) => (
            <SelectItem key={key} value={key}>
              {label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={filters.provider ?? "all"}
        onValueChange={(value) =>
          onFilterChange({
            ...filters,
            provider: value === "all" ? undefined : value,
          })
        }
      >
        <SelectTrigger className="w-[180px]">
          <SelectValue placeholder="Dostawca" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Wszyscy dostawcy</SelectItem>
          {SHIPMENT_PROVIDERS.map((provider) => (
            <SelectItem key={provider} value={provider}>
              {provider.toUpperCase()}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
