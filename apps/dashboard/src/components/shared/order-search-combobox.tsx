"use client";

import { useState, useEffect, useRef } from "react";
import { ChevronsUpDown, Check, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { cn, shortId, formatCurrency } from "@/lib/utils";
import { apiClient } from "@/lib/api-client";
import type { Order, ListResponse } from "@/types/api";

interface OrderSearchComboboxProps {
  value: string;
  onValueChange: (value: string) => void;
  disabled?: boolean;
}

export function OrderSearchCombobox({
  value,
  onValueChange,
  disabled,
}: OrderSearchComboboxProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [orders, setOrders] = useState<Order[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Fetch orders on search change with debounce
  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }

    if (!open) return;

    debounceRef.current = setTimeout(async () => {
      setIsLoading(true);
      try {
        const params = new URLSearchParams({ limit: "10" });
        if (search.trim()) {
          params.set("search", search.trim());
        }
        const data = await apiClient<ListResponse<Order>>(
          `/v1/orders?${params}`
        );
        setOrders(data.items ?? []);
      } catch {
        setOrders([]);
      } finally {
        setIsLoading(false);
      }
    }, 300);

    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [search, open]);

  // Load selected order details if value is set but no selectedOrder
  useEffect(() => {
    if (value && !selectedOrder) {
      apiClient<Order>(`/v1/orders/${value}`)
        .then((order) => setSelectedOrder(order))
        .catch(() => {});
    }
  }, [value, selectedOrder]);

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("pl-PL");
  };

  const getDisplayText = () => {
    if (selectedOrder) {
      return `${selectedOrder.customer_name} (${shortId(selectedOrder.id)}...)`;
    }
    if (value) {
      return `${shortId(value)}...`;
    }
    return "Wybierz zamówienie...";
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between font-normal"
          disabled={disabled}
        >
          <span className="truncate">{getDisplayText()}</span>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0" align="start">
        <Command shouldFilter={false}>
          <CommandInput
            placeholder="Szukaj zamówienia..."
            value={search}
            onValueChange={setSearch}
          />
          <CommandList>
            {isLoading ? (
              <div className="flex items-center justify-center py-6">
                <Loader2 className="h-4 w-4 animate-spin" />
              </div>
            ) : orders.length === 0 ? (
              <CommandEmpty>Nie znaleziono zamówień.</CommandEmpty>
            ) : (
              <CommandGroup>
                {orders.map((order) => (
                  <CommandItem
                    key={order.id}
                    value={order.id}
                    onSelect={() => {
                      onValueChange(order.id);
                      setSelectedOrder(order);
                      setOpen(false);
                    }}
                  >
                    <Check
                      className={cn(
                        "mr-2 h-4 w-4",
                        value === order.id ? "opacity-100" : "opacity-0"
                      )}
                    />
                    <div className="flex flex-col gap-0.5 overflow-hidden">
                      <span className="truncate font-medium">
                        {order.customer_name}
                      </span>
                      <span className="text-xs text-muted-foreground truncate">
                        {shortId(order.id)}... &middot;{" "}
                        {formatDate(order.created_at)} &middot;{" "}
                        {formatCurrency(order.total_amount, order.currency)}
                      </span>
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
