"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { Boxes, ChevronRight, PanelLeftClose, PanelLeftOpen } from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/lib/auth";
import { navItems, navGroups, type NavItem } from "@/lib/nav-items";
import { isNavItemActive } from "@/lib/nav-utils";
import { getVisibleNavItems } from "@/lib/readiness";
import { useSidebar } from "@/components/layout/sidebar-context";
import { useGroupExpansion } from "@/hooks/use-group-expansion";
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
  const { toggleGroup, isGroupExpanded } = useGroupExpansion();
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());
  const tNav = useTranslations("navigation");
  const tShared = useTranslations("shared.sidebar");

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

  const filteredItems = getVisibleNavItems(navItems, { isAdmin });

  const ungroupedItems = filteredItems.filter((item) => !item.group);
  const groupedItems = filteredItems.filter((item) => item.group);

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
                ? "bg-info/10 text-info"
                : "text-muted-foreground hover:bg-accent/60 hover:text-foreground"
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
          className={cn(
            "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
            isChild && "pl-9 py-1.5 text-[13px]",
            isActive
              ? "border-l-2 border-info bg-info/10 text-info shadow-sm"
              : "border-l-2 border-transparent text-muted-foreground hover:bg-accent/60 hover:text-foreground"
          )}
        >
          <item.icon className={cn("h-4 w-4 shrink-0", isChild && "h-3.5 w-3.5")} />
          <span className="truncate">{tNav(item.label)}</span>
        </Link>
      </div>
    );
  };

  const renderCollapsedNavLink = (item: NavItem) => {
    const isActive = isNavItemActive(pathname, item, false, filteredItems);

    return (
      <Tooltip key={item.href}>
        <TooltipTrigger asChild>
          <Link
            href={item.href}
            aria-label={tNav(item.label)}
            className={cn(
              "flex items-center justify-center rounded-md p-2 transition-colors",
              isActive
                ? "bg-info/10 text-info"
                : "text-muted-foreground hover:bg-accent/60 hover:text-foreground"
            )}
          >
            <item.icon className="h-4 w-4" />
          </Link>
        </TooltipTrigger>
        <TooltipContent side="right" className="font-medium">
          {tNav(item.label)}
          {item.children && item.children.length > 0 && (
            <div className="mt-1 space-y-0.5 text-xs text-muted-foreground">
              {item.children.map((child) => (
                <div key={child.href}>{tNav(child.label)}</div>
              ))}
            </div>
          )}
        </TooltipContent>
      </Tooltip>
    );
  };

  return (
    <aside
      className={cn(
        "hidden flex-col border-r bg-background transition-[width] duration-200 md:flex",
        collapsed ? "w-16" : "w-52"
      )}
    >
      {/* Header / Logo */}
      <div className="flex h-16 items-center border-b px-4">
        {collapsed ? (
          <Link
            href="/"
            aria-label={tNav("operationsDashboard")}
            className="flex w-full items-center justify-center"
          >
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-info/10 text-info">
              <Boxes className="h-5 w-5" aria-hidden="true" />
            </span>
          </Link>
        ) : (
          <Link href="/" className="flex items-center gap-3 font-semibold">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-info/10 text-info">
              <Boxes className="h-5 w-5" aria-hidden="true" />
            </span>
            <span className="text-lg tracking-normal text-foreground">OpenOMS</span>
          </Link>
        )}
      </div>

      {/* Navigation */}
      <nav
        aria-label={tShared("mainMenu")}
        className={cn(
          "flex-1 space-y-1 overflow-y-auto",
          collapsed ? "p-2" : "p-3"
        )}
      >
        {collapsed ? (
          <TooltipProvider delayDuration={0}>
            {ungroupedItems.map((item) => renderCollapsedNavLink(item))}
            {navGroups.map((group) => {
              const items = groupedItems.filter((item) => item.group === group.key);
              if (items.length === 0) return null;

              return (
                <div key={group.key}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="my-2 flex items-center justify-center">
                        <group.icon className="h-3.5 w-3.5 text-muted-foreground/50" />
                      </div>
                    </TooltipTrigger>
                    <TooltipContent side="right" className="font-medium">
                      {tNav(`groups.${group.label}`)}
                    </TooltipContent>
                  </Tooltip>
                  {items.map((item) => renderCollapsedNavLink(item))}
                </div>
              );
            })}
          </TooltipProvider>
        ) : (
          <>
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
                    className="flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent/50 hover:text-foreground"
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
                  aria-label={tShared("expandMenu")}
                  className="flex w-full items-center justify-center rounded-md p-2 text-muted-foreground transition-colors hover:bg-accent/60 hover:text-foreground"
                >
                  <PanelLeftOpen className="h-4 w-4" />
                </button>
              </TooltipTrigger>
              <TooltipContent side="right" className="font-medium">
                {tShared("expandMenu")}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ) : (
          <button
            onClick={toggleSidebar}
            aria-label={tShared("collapseMenu")}
            className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-xs text-muted-foreground transition-colors hover:bg-accent/60 hover:text-foreground"
          >
            <PanelLeftClose className="h-4 w-4" />
            <span>{tShared("collapseMenu")}</span>
            <kbd className="ml-auto rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px]">
              Ctrl+B
            </kbd>
          </button>
        )}
      </div>
    </aside>
  );
}
