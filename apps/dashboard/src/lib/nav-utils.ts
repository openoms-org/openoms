import type { NavItem } from "@/lib/nav-items";

/**
 * Determine if a nav item is the active route.
 *
 * Rules:
 *  - "/" is active only on exact match
 *  - Child items match on exact or prefix
 *  - Top-level items match on exact or prefix, but NOT if a more-specific
 *    sibling also matches (prevents /orders matching when on /orders/import)
 */
export function isNavItemActive(
  pathname: string,
  item: NavItem,
  isChild: boolean,
  allItems: NavItem[],
): boolean {
  if (isChild) {
    return pathname === item.href || pathname.startsWith(item.href + "/");
  }

  if (item.href === "/") {
    return pathname === "/";
  }

  if (pathname === item.href) return true;

  // Prefix match, but exclude if a more-specific sibling also matches
  if (pathname.startsWith(item.href + "/")) {
    const hasMoreSpecificSibling = allItems.some(
      (other) =>
        other.href !== item.href &&
        other.href.startsWith(item.href + "/") &&
        pathname.startsWith(other.href),
    );
    return !hasMoreSpecificSibling;
  }

  return false;
}
