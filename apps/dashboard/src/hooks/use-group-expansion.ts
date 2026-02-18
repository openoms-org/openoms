"use client";

import { useState, useEffect, useCallback } from "react";
import { usePathname } from "next/navigation";
import { navItems, navGroups } from "@/lib/nav-items";

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

export function useGroupExpansion() {
  const pathname = usePathname();
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(() => {
    return new Set(
      navGroups.filter((g) => g.defaultExpanded).map((g) => g.key)
    );
  });
  const [loaded, setLoaded] = useState(false);

  // Load from localStorage on mount
  useEffect(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        const parsed = JSON.parse(saved) as string[];
        setExpandedGroups(new Set(parsed));
      }
    } catch {
      // ignore
    }
    setLoaded(true);
  }, []);

  // Auto-expand the group containing the active route
  useEffect(() => {
    const activeGroup = findActiveGroup(pathname);
    if (activeGroup) {
      setExpandedGroups((prev) => {
        if (prev.has(activeGroup)) return prev;
        const next = new Set(prev);
        next.add(activeGroup);
        return next;
      });
    }
  }, [pathname]);

  // Persist to localStorage
  useEffect(() => {
    if (loaded) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify([...expandedGroups]));
    }
  }, [expandedGroups, loaded]);

  const toggleGroup = useCallback((groupKey: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(groupKey)) {
        next.delete(groupKey);
      } else {
        next.add(groupKey);
      }
      return next;
    });
  }, []);

  const isGroupExpanded = useCallback(
    (groupKey: string) => expandedGroups.has(groupKey),
    [expandedGroups]
  );

  return { toggleGroup, isGroupExpanded };
}
