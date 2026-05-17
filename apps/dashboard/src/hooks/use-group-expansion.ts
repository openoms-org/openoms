"use client";

import { useEffect, useCallback } from "react";
import { usePathname } from "next/navigation";
import { navItems, navGroups } from "@/lib/nav-items";
import { useHydratedState } from "@/hooks/use-effect-synced-state";

const STORAGE_KEY = "sidebar-expanded-groups";

function findActiveGroup(pathname: string): string | null {
  // Exact match first
  for (const item of navItems) {
    if (!item.group) continue;
    if (pathname === item.href) return item.group;
    if (item.children) {
      for (const child of item.children) {
        if (pathname === child.href || pathname.startsWith(child.href + "/")) {
          return item.group;
        }
      }
    }
  }
  // Prefix match for detail pages (e.g. /orders/[id])
  for (const item of navItems) {
    if (!item.group || item.href === "/") continue;
    if (pathname.startsWith(item.href + "/")) {
      return item.group;
    }
  }
  return null;
}

function defaultExpandedGroups(): Set<string> {
  return new Set(navGroups.filter((g) => g.defaultExpanded).map((g) => g.key));
}

export function useGroupExpansion() {
  const pathname = usePathname();
  const activeGroup = findActiveGroup(pathname);
  const readSavedGroups = useCallback(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        const parsed = JSON.parse(saved) as string[];
        return new Set(parsed);
      }
    } catch {
      // ignore
    }

    return defaultExpandedGroups();
  }, []);
  const [expandedGroups, setExpandedGroups, hydrated] = useHydratedState(
    defaultExpandedGroups(),
    readSavedGroups,
  );

  // Persist to localStorage
  useEffect(() => {
    if (hydrated) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify([...expandedGroups]));
    }
  }, [expandedGroups, hydrated]);

  const toggleGroup = useCallback(
    (groupKey: string) => {
      setExpandedGroups((prev) => {
        const next = new Set(prev);
        if (next.has(groupKey)) {
          next.delete(groupKey);
        } else {
          next.add(groupKey);
        }
        return next;
      });
    },
    [setExpandedGroups]
  );

  const isGroupExpanded = useCallback(
    (groupKey: string) => expandedGroups.has(groupKey) || activeGroup === groupKey,
    [activeGroup, expandedGroups]
  );

  return { toggleGroup, isGroupExpanded };
}
