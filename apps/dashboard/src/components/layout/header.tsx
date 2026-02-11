"use client";

import { Breadcrumbs } from "./breadcrumbs";
import { UserMenu } from "./user-menu";
import { ThemeToggle } from "./theme-toggle";
import { MobileNav } from "./mobile-nav";
import { Separator } from "@/components/ui/separator";
import { ConnectionStatus } from "./connection-status";

export function Header() {
  return (
    <header className="flex h-14 items-center gap-4 border-b bg-background px-4">
      <MobileNav />
      <Separator orientation="vertical" className="h-6 md:hidden" />
      <div className="flex-1">
        <Breadcrumbs />
      </div>
      <div className="flex items-center gap-2">
        <ConnectionStatus />
        <ThemeToggle />
        <UserMenu />
      </div>
    </header>
  );
}
