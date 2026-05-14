import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "@/lib/auth";

const { redirect } = vi.hoisted(() => ({
  redirect: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  redirect,
}));

vi.mock("next-intl", () => ({
  useTranslations: (namespace?: string) => (key: string) => {
    const messages: Record<string, string> = {
      "settings.index.title": "Ustawienia",
      "settings.index.description":
        "Wybierz obszar konfiguracji OpenOMS.",
      "settings.index.sectionCount": "3 sekcje",
      "settings.index.openSection": "Otwórz",
      "navigation.company": "Firma",
      "navigation.users": "Użytkownicy",
      "navigation.roles": "Role",
      "navigation.security": "Bezpieczeństwo",
      "navigation.groups.settings": "Ustawienia",
    };

    return messages[namespace ? `${namespace}.${key}` : key] ?? key;
  },
}));

import SettingsIndexPage from "./page";

const makeUser = (role: "owner" | "admin" | "member") => ({
  id: `${role}-user`,
  tenant_id: "tenant-1",
  email: `${role}@example.com`,
  name: `${role} user`,
  role,
  created_at: "2026-05-14T00:00:00Z",
  updated_at: "2026-05-14T00:00:00Z",
});

describe("SettingsIndexPage", () => {
  beforeEach(() => {
    redirect.mockClear();
    useAuthStore.setState({
      user: makeUser("admin"),
    });
  });

  it("renders a settings overview instead of redirecting to security", () => {
    render(<SettingsIndexPage />);

    expect(redirect).not.toHaveBeenCalled();
    expect(
      screen.getByRole("heading", { level: 1, name: "Ustawienia" })
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Firma/i })).toHaveAttribute(
      "href",
      "/settings/company"
    );
    expect(screen.getByRole("link", { name: /Bezpieczeństwo/i })).toHaveAttribute(
      "href",
      "/settings/security"
    );
  });

  it("keeps admin-only settings out of the overview for regular users", () => {
    useAuthStore.setState({
      user: makeUser("member"),
    });

    render(<SettingsIndexPage />);

    expect(screen.getByRole("link", { name: /Bezpieczeństwo/i })).toHaveAttribute(
      "href",
      "/settings/security"
    );
    expect(screen.queryByRole("link", { name: /Firma/i })).not.toBeInTheDocument();
  });
});
