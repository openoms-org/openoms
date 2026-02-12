"use client";

import { useState, useEffect, useRef } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { ORDER_STATUSES, ORDER_SOURCES, PAYMENT_STATUSES, ORDER_SOURCE_LABELS, ORDER_PRIORITIES } from "@/lib/constants";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";

interface OrderFilters {
  status?: string;
  source?: string;
  search?: string;
  payment_status?: string;
  tag?: string;
  priority?: string;
}

interface OrderFiltersProps {
  filters: OrderFilters;
  onFilterChange: (filters: OrderFilters) => void;
}

export function OrderFilters({ filters, onFilterChange }: OrderFiltersProps) {
  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;

  const [localSearch, setLocalSearch] = useState(filters.search || "");
  const [localTag, setLocalTag] = useState(filters.tag || "");
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const tagDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  // Sync local search when filters.search changes externally (e.g. reset)
  useEffect(() => {
    setLocalSearch(filters.search || "");
  }, [filters.search]);

  // Sync local tag when filters.tag changes externally (e.g. reset)
  useEffect(() => {
    setLocalTag(filters.tag || "");
  }, [filters.tag]);

  const handleSearchChange = (value: string) => {
    setLocalSearch(value);
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    debounceRef.current = setTimeout(() => {
      onFilterChange({ ...filters, search: value || undefined });
    }, 300);
  };

  const handleTagChange = (value: string) => {
    setLocalTag(value);
    if (tagDebounceRef.current) {
      clearTimeout(tagDebounceRef.current);
    }
    tagDebounceRef.current = setTimeout(() => {
      onFilterChange({ ...filters, tag: value || undefined });
    }, 300);
  };

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
      if (tagDebounceRef.current) {
        clearTimeout(tagDebounceRef.current);
      }
    };
  }, []);

  return (
    <div className="flex flex-wrap items-center gap-4">
      <Input
        placeholder="Szukaj klienta (imię, email, telefon)..."
        value={localSearch}
        onChange={(e) => handleSearchChange(e.target.value)}
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
                {ORDER_SOURCE_LABELS[source] || source}
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
      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">Priorytet:</span>
        <Select
          value={filters.priority || "all"}
          onValueChange={(v) =>
            onFilterChange({ ...filters, priority: v === "all" ? undefined : v })
          }
        >
          <SelectTrigger className="w-[160px]" size="sm">
            <SelectValue placeholder="Wszystkie" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Wszystkie</SelectItem>
            {Object.entries(ORDER_PRIORITIES).map(([key, { label }]) => (
              <SelectItem key={key} value={key}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <Input
        placeholder="Filtruj po tagu..."
        value={localTag}
        onChange={(e) => handleTagChange(e.target.value)}
        className="w-[180px]"
      />
    </div>
  );
}
