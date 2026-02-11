"use client";

import { useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandSeparator,
} from "@/components/ui/command";
import {
  ShoppingCart,
  Package,
  Truck,
  Contact,
} from "lucide-react";
import { navItems } from "@/lib/nav-items";
import { useAuthStore } from "@/lib/auth";

interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const quickActions = [
  { label: "Nowe zamówienie", href: "/orders/new", icon: ShoppingCart },
  { label: "Nowy produkt", href: "/products/new", icon: Package },
  { label: "Nowa przesylka", href: "/shipments/new", icon: Truck },
  { label: "Nowy klient", href: "/customers/new", icon: Contact },
];

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.role === "admin" || user?.role === "owner";

  const filteredNavItems = navItems.filter(
    (item) => !item.adminOnly || isAdmin
  );

  const handleSelect = useCallback(
    (href: string) => {
      onOpenChange(false);
      router.push(href);
    },
    [router, onOpenChange]
  );

  return (
    <CommandDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Paleta polecen"
      description="Wyszukaj strone lub akcje..."
    >
      <CommandInput placeholder="Szukaj..." />
      <CommandList>
        <CommandEmpty>Nie znaleziono wyników.</CommandEmpty>

        <CommandGroup heading="Nawigacja">
          {filteredNavItems.map((item) => (
            <CommandItem
              key={item.href}
              value={`${item.label} ${item.group || ""}`}
              onSelect={() => handleSelect(item.href)}
            >
              <item.icon className="mr-2 h-4 w-4" />
              <span>{item.label}</span>
              {item.group && (
                <span className="ml-auto text-xs text-muted-foreground">
                  {item.group}
                </span>
              )}
            </CommandItem>
          ))}
        </CommandGroup>

        <CommandSeparator />

        <CommandGroup heading="Akcje">
          {quickActions.map((action) => (
            <CommandItem
              key={action.href}
              value={action.label}
              onSelect={() => handleSelect(action.href)}
            >
              <action.icon className="mr-2 h-4 w-4" />
              <span>{action.label}</span>
            </CommandItem>
          ))}
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}
