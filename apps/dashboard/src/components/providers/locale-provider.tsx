"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/lib/auth";

export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const user = useAuthStore((s) => s.user);

  useEffect(() => {
    if (user?.language) {
      const currentCookie = document.cookie
        .split("; ")
        .find((c) => c.startsWith("NEXT_LOCALE="))
        ?.split("=")[1];

      if (currentCookie !== user.language) {
        document.cookie = `NEXT_LOCALE=${user.language}; path=/; max-age=31536000; SameSite=Lax`;
        window.location.reload();
      }
    }
  }, [user?.language]);

  return <>{children}</>;
}
