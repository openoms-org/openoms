"use client";

import { Breadcrumbs } from "./breadcrumbs";
import { UserMenu } from "./user-menu";
import { MobileNav } from "./mobile-nav";
import { Separator } from "@/components/ui/separator";

export function Header() {
  return (
    <header className="flex h-14 items-center gap-4 border-b bg-background px-4">
      <MobileNav />
      <Separator orientation="vertical" className="h-6 md:hidden" />
      <div className="flex-1">
        <Breadcrumbs />
      </div>
      <div className="flex items-center gap-2">
        {/* ThemeToggle hidden — dark mode not fully implemented */}
        <UserMenu />
      </div>
    </header>
  );
}
