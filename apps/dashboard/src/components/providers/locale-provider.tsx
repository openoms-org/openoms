"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/lib/auth";

export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const user = useAuthStore((s) => s.user);

  useEffect(() => {
    // Skip if user just changed locale via LanguageSelector (prevents loop)
    if (sessionStorage.getItem("locale-changing") === "1") {
      sessionStorage.removeItem("locale-changing");
      return;
    }

    if (user?.language) {
      const currentCookie = document.cookie
        .split("; ")
        .find((c) => c.startsWith("NEXT_LOCALE="))
        ?.split("=")[1];

      if (currentCookie !== user.language) {
        document.cookie = `NEXT_LOCALE=${user.language}; path=/; max-age=31536000; SameSite=Lax`;
      }
    }
  }, [user?.language]);

  return <>{children}</>;
}
