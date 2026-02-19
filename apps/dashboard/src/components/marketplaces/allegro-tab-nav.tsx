"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const ALLEGRO_TABS = [
  { href: "/marketplaces/allegro/offers", label: "Oferty" },
  { href: "/marketplaces/allegro/catalog", label: "Katalog" },
  { href: "/marketplaces/allegro/promotions", label: "Promocje" },
  { href: "/marketplaces/allegro/messages", label: "Wiadomości" },
  { href: "/marketplaces/allegro/returns", label: "Zwroty" },
  { href: "/marketplaces/allegro/disputes", label: "Spory" },
  { href: "/marketplaces/allegro/delivery", label: "Dostawa" },
  { href: "/marketplaces/allegro/policies", label: "Polityki" },
  { href: "/marketplaces/allegro/finance", label: "Finanse" },
  { href: "/marketplaces/allegro/ratings", label: "Oceny" },
  { href: "/marketplaces/allegro/shipments", label: "Przesyłki" },
] as const;

export function AllegroTabNav() {
  const pathname = usePathname();

  return (
    <div className="overflow-x-auto">
      <nav className="flex border-b" role="tablist">
        {ALLEGRO_TABS.map((tab) => {
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
