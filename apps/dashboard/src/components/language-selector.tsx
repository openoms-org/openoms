"use client";

import { useTranslations } from "next-intl";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAuthStore } from "@/lib/auth";
import { apiClient } from "@/lib/api-client";
import { toast } from "sonner";

const LANGUAGES = [
  { code: "en", label: "English", flag: "EN" },
  { code: "pl", label: "Polski", flag: "PL" },
];

export function LanguageSelector({ compact }: { compact?: boolean }) {
  const t = useTranslations("settings");
  const user = useAuthStore((s) => s.user);
  const locale = useAuthStore((s) => s.locale);

  const handleChange = async (newLocale: string) => {
    try {
      if (user) {
        await apiClient("/v1/users/me", {
          method: "PATCH",
          body: JSON.stringify({ language: newLocale }),
        });
      }
      sessionStorage.setItem("locale-changing", "1");
      document.cookie = `NEXT_LOCALE=${newLocale}; path=/; max-age=31536000; SameSite=Lax`;
      window.location.reload();
    } catch {
      toast.error(t("languageSaveFailed"));
    }
  };

  return (
    <Select value={locale} onValueChange={handleChange}>
      <SelectTrigger className={compact ? "w-[70px]" : "w-[140px]"}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {LANGUAGES.map((lang) => (
          <SelectItem key={lang.code} value={lang.code}>
            {compact ? lang.flag : `${lang.flag} ${lang.label}`}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
