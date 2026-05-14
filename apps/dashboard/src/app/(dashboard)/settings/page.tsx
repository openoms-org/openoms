"use client";

import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { useTranslations } from "next-intl";
import { PageHeader } from "@/components/shared/page-header";
import { Surface } from "@/components/shared/surface";
import { Button } from "@/components/ui/button";
import { navGroups, navItems, type NavItem } from "@/lib/nav-items";
import { getVisibleNavItems } from "@/lib/readiness";
import { useAuthStore } from "@/lib/auth";
import { cn } from "@/lib/utils";

function isSettingsSection(item: NavItem): boolean {
  return item.href.startsWith("/settings/") || item.group === "settings";
}

function uniqueByHref(items: NavItem[]): NavItem[] {
  const seen = new Set<string>();

  return items.filter((item) => {
    if (seen.has(item.href)) {
      return false;
    }
    seen.add(item.href);
    return true;
  });
}

function groupSettingsSections(items: NavItem[]) {
  const orderedGroups = navGroups.map((group) => group.key);
  const fallbackGroup = "settings";
  const groups = new Map<string, NavItem[]>();

  for (const item of items) {
    const group = item.group ?? fallbackGroup;
    groups.set(group, [...(groups.get(group) ?? []), item]);
  }

  return [...groups.entries()].sort(([groupA], [groupB]) => {
    const indexA = orderedGroups.indexOf(groupA);
    const indexB = orderedGroups.indexOf(groupB);
    const orderA = indexA === -1 ? orderedGroups.length : indexA;
    const orderB = indexB === -1 ? orderedGroups.length : indexB;
    return orderA - orderB;
  });
}

export default function SettingsIndexPage() {
  const t = useTranslations("settings.index");
  const tNav = useTranslations("navigation");
  const user = useAuthStore((state) => state.user);
  const isAdmin = user?.role === "admin" || user?.role === "owner";
  const visibleSettings = uniqueByHref(
    getVisibleNavItems(navItems, { isAdmin }).filter(isSettingsSection)
  );
  const groupedSettings = groupSettingsSections(visibleSettings);

  return (
    <div className="space-y-6">
      <PageHeader
        title={t("title")}
        description={t("description")}
        meta={t("sectionCount", { count: visibleSettings.length })}
      />

      {visibleSettings.length === 0 ? (
        <Surface className="text-sm text-muted-foreground">
          {t("emptyState")}
        </Surface>
      ) : (
        <div className="space-y-7">
          {groupedSettings.map(([group, items]) => (
            <section key={group} className="space-y-3">
              <h2 className="text-sm font-semibold text-muted-foreground">
                {tNav(`groups.${group}`)}
              </h2>
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {items.map((item) => {
                  const Icon = item.icon;

                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      aria-label={tNav(item.label)}
                      className={cn(
                        "group rounded-lg border bg-card p-4 text-card-foreground transition-colors",
                        "hover:border-info/40 hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      )}
                    >
                      <div className="flex items-start justify-between gap-4">
                        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-info/10 text-info">
                          <Icon className="h-5 w-5" aria-hidden="true" />
                        </span>
                        <Button
                          asChild
                          variant="ghost"
                          size="sm"
                          className="pointer-events-none h-8 px-2 text-muted-foreground group-hover:text-foreground"
                          tabIndex={-1}
                        >
                          <span>
                            {t("openSection")}
                            <ArrowUpRight className="ml-1 h-3.5 w-3.5" aria-hidden="true" />
                          </span>
                        </Button>
                      </div>
                      <div className="mt-4 space-y-1">
                        <h3 className="text-sm font-semibold text-foreground">
                          {tNav(item.label)}
                        </h3>
                        <p className="text-xs text-muted-foreground">{item.href}</p>
                      </div>
                    </Link>
                  );
                })}
              </div>
            </section>
          ))}
        </div>
      )}
    </div>
  );
}
