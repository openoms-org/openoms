"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Package } from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/lib/auth";
import { navItems } from "@/lib/nav-items";

export function Sidebar() {
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

  const renderNavLink = (item: (typeof filteredItems)[number]) => {
    const isActive =
      item.href === "/"
        ? pathname === "/"
        : pathname.startsWith(item.href);

    return (
      <Link
        key={item.href}
        href={item.href}
        className={cn(
          "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
          isActive
            ? "bg-sidebar-accent text-sidebar-accent-foreground"
            : "text-sidebar-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground"
        )}
      >
        <item.icon className="h-4 w-4" />
        {item.label}
      </Link>
    );
  };

  return (
    <aside className="hidden md:flex w-64 flex-col border-r bg-sidebar">
      <div className="flex h-14 items-center border-b px-4">
        <Link href="/" className="flex items-center gap-2 font-semibold">
          <Package className="h-6 w-6" />
          <span>OpenOMS</span>
        </Link>
      </div>
      <nav className="flex-1 space-y-1 overflow-y-auto p-3">
        {ungroupedItems.map(renderNavLink)}

        {groupOrder.map((group) => (
          <div key={group}>
            <p className="text-xs font-semibold uppercase text-muted-foreground mt-4 mb-1 px-3">
              {group}
            </p>
            {groupedItems
              .filter((item) => item.group === group)
              .map(renderNavLink)}
          </div>
        ))}
      </nav>
    </aside>
  );
}
