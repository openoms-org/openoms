"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { ChevronRight, Menu, Package } from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/lib/auth";
import { navItems, navGroups, type NavItem } from "@/lib/nav-items";
import { isNavItemActive } from "@/lib/nav-utils";
import { getVisibleNavItems } from "@/lib/readiness";
import { useGroupExpansion } from "@/hooks/use-group-expansion";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";

export function MobileNav() {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.role === "admin" || user?.role === "owner";
  const { toggleGroup, isGroupExpanded } = useGroupExpansion();
  const tNav = useTranslations("navigation");
  const tShared = useTranslations("shared");

  const filteredItems = getVisibleNavItems(navItems, { isAdmin });

  const ungroupedItems = filteredItems.filter((item) => !item.group);
  const groupedItems = filteredItems.filter((item) => item.group);

  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());

  const toggleExpand = (href: string) => {
    setExpandedItems((prev) => {
      const next = new Set(prev);
      if (next.has(href)) {
        next.delete(href);
      } else {
        next.add(href);
      }
      return next;
    });
  };

  const renderNavLink = (item: NavItem, isChild = false) => {
    const isActive = isNavItemActive(pathname, item, isChild, filteredItems);

    const hasChildren = !isChild && item.children && item.children.length > 0;
    const isExpanded = hasChildren && (expandedItems.has(item.href) || pathname.startsWith(item.href));

    if (hasChildren) {
      return (
        <div key={item.href}>
          <button
            onClick={() => toggleExpand(item.href)}
            aria-expanded={isExpanded}
            className={cn(
              "flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
              isActive
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground"
            )}
          >
            <item.icon className="h-4 w-4" />
            {tNav(item.label)}
            <ChevronRight
              className={cn(
                "ml-auto h-3 w-3 transition-transform duration-200",
                isExpanded && "rotate-90"
              )}
            />
          </button>
          {isExpanded &&
            item.children!.map((child) => renderNavLink(child, true))}
        </div>
      );
    }

    return (
      <div key={item.href}>
        <Link
          href={item.href}
          onClick={() => setOpen(false)}
          className={cn(
            "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
            isChild && "pl-9 py-1.5 text-[13px]",
            isActive
              ? "border-l-2 border-primary bg-accent text-accent-foreground"
              : "border-l-2 border-transparent text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground"
          )}
        >
          <item.icon className={cn("h-4 w-4 shrink-0", isChild && "h-3.5 w-3.5")} />
          <span className="truncate">{tNav(item.label)}</span>
        </Link>
      </div>
    );
  };

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" className="md:hidden">
          <Menu className="h-5 w-5" />
          <span className="sr-only">{tShared("mobileNav.openMenu")}</span>
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-64 p-0">
        <SheetHeader className="border-b px-4 py-3">
          <SheetTitle className="flex items-center gap-2 text-left">
            <Package className="h-6 w-6" />
            OpenOMS
          </SheetTitle>
        </SheetHeader>
        <nav aria-label={tShared("sidebar.mainMenu")} className="space-y-1 overflow-y-auto p-3">
          {ungroupedItems.map((item) => renderNavLink(item))}

          {navGroups.map((group) => {
            const items = groupedItems.filter((item) => item.group === group.key);
            if (items.length === 0) return null;
            const expanded = isGroupExpanded(group.key);

            return (
              <div key={group.key} className="mt-3 first:mt-0">
                <button
                  onClick={() => toggleGroup(group.key)}
                  aria-expanded={expanded}
                  className="flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-accent/30 hover:text-accent-foreground transition-colors"
                >
                  <ChevronRight
                    className={cn(
                      "h-3 w-3 shrink-0 transition-transform duration-200",
                      expanded && "rotate-90"
                    )}
                  />
                  <group.icon className="h-3.5 w-3.5 shrink-0" />
                  <span className="truncate">{tNav(`groups.${group.label}`)}</span>
                </button>
                {expanded && (
                  <div className="mt-0.5 space-y-0.5">
                    {items.map((item) => renderNavLink(item))}
                  </div>
                )}
              </div>
            );
          })}
        </nav>
      </SheetContent>
    </Sheet>
  );
}
