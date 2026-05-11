"use client";

import { Breadcrumbs } from "./breadcrumbs";
import { UserMenu } from "./user-menu";
import { ThemeToggle } from "./theme-toggle";
import { MobileNav } from "./mobile-nav";
import { Separator } from "@/components/ui/separator";
import { ConnectionStatus } from "./connection-status";
import { Search } from "lucide-react";
import { useTranslations } from "next-intl";

interface HeaderProps {
  onOpenCommandPalette: () => void;
}

export function Header({ onOpenCommandPalette }: HeaderProps) {
  const t = useTranslations("commandPalette");

  return (
    <header className="flex h-16 items-center gap-4 border-b bg-background/95 px-4 backdrop-blur">
      <MobileNav />
      <Separator orientation="vertical" className="h-6 md:hidden" />
      <div className="hidden min-w-0 flex-1 items-center gap-4 md:flex">
        <Breadcrumbs />
        <button
          type="button"
          onClick={onOpenCommandPalette}
          className="mx-auto flex h-10 w-full max-w-[560px] items-center gap-3 rounded-md border bg-background px-3 text-left text-sm text-muted-foreground shadow-sm transition-colors hover:border-primary/30 hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <Search className="h-4 w-4 shrink-0" aria-hidden="true" />
          <span className="min-w-0 flex-1 truncate">{t("placeholder")}</span>
          <kbd className="hidden rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground lg:inline-flex">
            Ctrl K
          </kbd>
        </button>
      </div>
      <button
        type="button"
        onClick={onOpenCommandPalette}
        aria-label={t("title")}
        className="flex h-10 w-10 items-center justify-center rounded-md border bg-background text-muted-foreground shadow-sm md:hidden"
      >
        <Search className="h-4 w-4" aria-hidden="true" />
      </button>
      <div className="flex items-center gap-2">
        <ConnectionStatus />
        <ThemeToggle />
        <UserMenu />
      </div>
    </header>
  );
}
