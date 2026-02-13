"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronRight, Menu, Package } from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/lib/auth";
import { navItems, type NavItem } from "@/lib/nav-items";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";

export function MobileNav() {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.role === "admin" || user?.role === "owner";

  const filteredItems = navItems.filter(
    (item) => !item.adminOnly || isAdmin
  );

  // Separate ungrouped items (Dashboard) from grouped items
  const ungroupedItems = filteredItems.filter((item) => !item.group);
  const groupedItems = filteredItems.filter((item) => item.group);

  // Preserve order: collect groups in the order they first appear
  const groupOrder: string[] = [];
  for (const item of groupedItems) {
    if (item.group && !groupOrder.includes(item.group)) {
      groupOrder.push(item.group);
    }
  }

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
    const isActive = isChild
      ? pathname === item.href || pathname.startsWith(item.href + "/")
      : item.href === "/"
        ? pathname === "/"
        : pathname === item.href || (pathname.startsWith(item.href + "/") && !filteredItems.some((other) => other.href !== item.href && other.href.startsWith(item.href + "/") && pathname.startsWith(other.href)));

    const hasChildren = !isChild && item.children && item.children.length > 0;
    const isExpanded = hasChildren && (expandedItems.has(item.href) || pathname.startsWith(item.href));

    if (hasChildren) {
      return (
        <div key={item.href}>
          <button
            onClick={() => toggleExpand(item.href)}
            className={cn(
              "flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
              isActive
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground"
            )}
          >
            <item.icon className="h-4 w-4" />
            {item.label}
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
              ? "bg-accent text-accent-foreground"
              : "text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground"
          )}
        >
          <item.icon className={cn("h-4 w-4", isChild && "h-3.5 w-3.5")} />
          {item.label}
        </Link>
      </div>
    );
  };

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" className="md:hidden">
          <Menu className="h-5 w-5" />
          <span className="sr-only">Otwórz menu</span>
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-64 p-0">
        <SheetHeader className="border-b px-4 py-3">
          <SheetTitle className="flex items-center gap-2 text-left">
            <Package className="h-6 w-6" />
            OpenOMS
          </SheetTitle>
        </SheetHeader>
        <nav className="space-y-1 overflow-y-auto p-3">
          {ungroupedItems.map((item) => renderNavLink(item))}

          {groupOrder.map((group) => (
            <div key={group}>
              <p className="text-xs font-semibold uppercase text-muted-foreground mt-4 mb-1 px-3">
                {group}
              </p>
              {groupedItems
                .filter((item) => item.group === group)
                .map((item) => renderNavLink(item))}
            </div>
          ))}
        </nav>
      </SheetContent>
    </Sheet>
  );
}
