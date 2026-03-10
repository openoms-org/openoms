"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";

const ALLEGRO_TABS = [
  { href: "/marketplaces/allegro/offers", labelKey: "tabs.offers" },
  { href: "/marketplaces/allegro/catalog", labelKey: "tabs.catalog" },
  { href: "/marketplaces/allegro/promotions", labelKey: "tabs.promotions" },
  { href: "/marketplaces/allegro/messages", labelKey: "tabs.messages" },
  { href: "/marketplaces/allegro/returns", labelKey: "tabs.returns" },
  { href: "/marketplaces/allegro/disputes", labelKey: "tabs.disputes" },
  { href: "/marketplaces/allegro/delivery", labelKey: "tabs.delivery" },
  { href: "/marketplaces/allegro/policies", labelKey: "tabs.policies" },
  { href: "/marketplaces/allegro/finance", labelKey: "tabs.finance" },
  { href: "/marketplaces/allegro/ratings", labelKey: "tabs.ratings" },
  { href: "/marketplaces/allegro/shipments", labelKey: "tabs.shipments" },
] as const;

export function AllegroTabNav() {
  const pathname = usePathname();
  const t = useTranslations("marketplaces");

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
              {t(tab.labelKey)}
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
