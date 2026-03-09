"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const EBAY_TABS = [
  { href: "/marketplaces/ebay", label: "Konfiguracja" },
  { href: "/marketplaces/ebay/offers", label: "Oferty" },
] as const;

export function EbayTabNav() {
  const pathname = usePathname();

  return (
    <div className="overflow-x-auto">
      <nav className="flex border-b" role="tablist">
        {EBAY_TABS.map((tab) => {
          const isActive = pathname === tab.href;
          return (
            <Link
              key={tab.href}
              href={tab.href}
              role="tab"
              aria-selected={isActive}
              className={cn(
                "shrink-0 px-4 py-2.5 text-sm font-medium transition-colors whitespace-nowrap",
                "hover:text-foreground",
                isActive
                  ? "border-b-2 border-primary text-foreground"
                  : "text-muted-foreground"
              )}
            >
              {tab.label}
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
