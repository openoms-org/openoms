"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronRight, Package, PanelLeftClose, PanelLeftOpen } from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/lib/auth";
import { navItems, type NavItem } from "@/lib/nav-items";
import { useSidebar } from "@/components/layout/sidebar-context";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export function Sidebar() {
  const pathname = usePathname();
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.role === "admin" || user?.role === "owner";
  const { collapsed, toggleSidebar } = useSidebar();
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());

  const toggleExpand = useCallback((href: string) => {
    setExpandedItems((prev) => {
      const next = new Set(prev);
      if (next.has(href)) {
        next.delete(href);
      } else {
        next.add(href);
      }
      return next;
    });
  }, []);

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

  const renderNavLink = (item: NavItem, isChild = false) => {
    const isActive = isChild
      ? pathname === item.href || pathname.startsWith(item.href + "/")
      : item.href === "/"
        ? pathname === "/"
        : pathname === item.href || (pathname.startsWith(item.href + "/") && !filteredItems.some((other) => other.href !== item.href && other.href.startsWith(item.href + "/") && pathname.startsWith(other.href)));

    const hasChildren = !isChild && item.children && item.children.length > 0;
    // Expanded if manually toggled OR if pathname matches a child route
    const isExpanded = hasChildren && (expandedItems.has(item.href) || pathname.startsWith(item.href));

    // Items with children: button toggles expand/collapse, no navigation
    if (hasChildren) {
      return (
        <div key={item.href}>
          <button
            onClick={() => toggleExpand(item.href)}
            className={cn(
              "flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
              isActive
                ? "bg-sidebar-accent text-sidebar-accent-foreground"
                : "text-sidebar-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground"
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
          className={cn(
            "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
            isChild && "pl-9 py-1.5 text-[13px]",
            isActive
              ? "bg-sidebar-accent text-sidebar-accent-foreground"
              : "text-sidebar-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground"
          )}
        >
          <item.icon className={cn("h-4 w-4", isChild && "h-3.5 w-3.5")} />
          {item.label}
        </Link>
      </div>
    );
  };

  const renderCollapsedNavLink = (item: NavItem) => {
    const isActive =
      item.href === "/"
        ? pathname === "/"
        : pathname === item.href || (pathname.startsWith(item.href + "/") && !filteredItems.some((other) => other.href !== item.href && other.href.startsWith(item.href + "/") && pathname.startsWith(other.href)));

    const hasChildren = item.children && item.children.length > 0;

    return (
      <TooltipProvider key={item.href} delayDuration={0}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Link
              href={item.href}
              className={cn(
                "flex items-center justify-center rounded-md p-2 transition-colors",
                isActive
                  ? "bg-sidebar-accent text-sidebar-accent-foreground"
                  : "text-sidebar-foreground hover:bg-sidebar-accent/50"
              )}
            >
              <item.icon className="h-4 w-4" />
            </Link>
          </TooltipTrigger>
          <TooltipContent side="right" className="font-medium">
            {item.label}
            {hasChildren && item.children && (
              <div className="mt-1 space-y-0.5 text-xs text-muted-foreground">
                {item.children.map((child) => (
                  <div key={child.href}>{child.label}</div>
                ))}
              </div>
            )}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  };

  return (
    <aside
      className={cn(
        "hidden md:flex flex-col border-r bg-sidebar transition-[width] duration-200",
        collapsed ? "w-14" : "w-64"
      )}
    >
      {/* Header / Logo */}
      <div className="flex h-14 items-center border-b px-4">
        {collapsed ? (
          <Link href="/" className="flex w-full items-center justify-center">
            <Package className="h-6 w-6" />
          </Link>
        ) : (
          <Link href="/" className="flex items-center gap-2 font-semibold">
            <Package className="h-6 w-6" />
            <span>OpenOMS</span>
          </Link>
        )}
      </div>

      {/* Navigation */}
      <nav
        className={cn(
          "flex-1 space-y-1 overflow-y-auto",
          collapsed ? "p-2" : "p-3"
        )}
      >
        {collapsed ? (
          <>
            {ungroupedItems.map((item) => renderCollapsedNavLink(item))}
            {groupOrder.map((group) => (
              <div key={group}>
                <div className="my-2 border-t" />
                {groupedItems
                  .filter((item) => item.group === group)
                  .map((item) => renderCollapsedNavLink(item))}
              </div>
            ))}
          </>
        ) : (
          <>
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
          </>
        )}
      </nav>

      {/* Footer toggle */}
      <div className="border-t px-2 py-3">
        {collapsed ? (
          <TooltipProvider delayDuration={0}>
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  onClick={toggleSidebar}
                  className="flex w-full items-center justify-center rounded-md p-2 text-muted-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground transition-colors"
                >
                  <PanelLeftOpen className="h-4 w-4" />
                </button>
              </TooltipTrigger>
              <TooltipContent side="right" className="font-medium">
                Rozwiń menu (Ctrl+B)
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ) : (
          <button
            onClick={toggleSidebar}
            className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-xs text-muted-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground transition-colors"
          >
            <PanelLeftClose className="h-4 w-4" />
            <span>Zwiń menu</span>
            <kbd className="ml-auto rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px]">
              Ctrl+B
            </kbd>
          </button>
        )}
      </div>
    </aside>
  );
}
