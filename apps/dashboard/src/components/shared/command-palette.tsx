"use client";

import { useCallback } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
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
import { navItems, flattenNavItems } from "@/lib/nav-items";
import { useAuthStore } from "@/lib/auth";
import { getVisibleNavItems, isRouteAccessible } from "@/lib/readiness";

interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const quickActions = [
  { labelKey: "newOrder" as const, href: "/orders/new", icon: ShoppingCart },
  { labelKey: "newProduct" as const, href: "/products/new", icon: Package },
  { labelKey: "newShipment" as const, href: "/shipments/new", icon: Truck },
  { labelKey: "newCustomer" as const, href: "/customers/new", icon: Contact },
];

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.role === "admin" || user?.role === "owner";
  const t = useTranslations("commandPalette");
  const tNav = useTranslations("navigation");

  const allNavItems = flattenNavItems(getVisibleNavItems(navItems, { isAdmin }));
  const visibleQuickActions = quickActions.filter((action) =>
    isRouteAccessible(action.href),
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
      title={t("title")}
      description={t("placeholder")}
    >
      <CommandInput placeholder={t("placeholder")} />
      <CommandList>
        <CommandEmpty>{t("noResults")}</CommandEmpty>

        <CommandGroup heading={t("navigation")}>
          {allNavItems.map((item) => (
            <CommandItem
              key={item.href}
              value={`${tNav(item.label)} ${item.group ? tNav(`groups.${item.group}`) : ""}`}
              onSelect={() => handleSelect(item.href)}
            >
              <item.icon className="mr-2 h-4 w-4" />
              <span>{tNav(item.label)}</span>
              {item.group && (
                <span className="ml-auto text-xs text-muted-foreground">
                  {tNav(`groups.${item.group}`)}
                </span>
              )}
            </CommandItem>
          ))}
        </CommandGroup>

        <CommandSeparator />

        <CommandGroup heading={t("quickActions")}>
          {visibleQuickActions.map((action) => (
            <CommandItem
              key={action.href}
              value={t(action.labelKey)}
              onSelect={() => handleSelect(action.href)}
            >
              <action.icon className="mr-2 h-4 w-4" />
              <span>{t(action.labelKey)}</span>
            </CommandItem>
          ))}
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}
