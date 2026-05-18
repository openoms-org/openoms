import { type ReactNode } from "react";
import { describe, expect, it, beforeEach, vi } from "vitest";
import { render, screen, waitFor, userEvent } from "@/test/test-utils";
import LoginPage from "./page";

const loginMock = vi.fn();
const verify2FALoginMock = vi.fn();

vi.mock("next/link", () => ({
  default: ({ children, href, ...props }: { children: ReactNode; href: string }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => {
    const messages: Record<string, string> = {
      "login.title": "Zaloguj się do panelu",
      "login.subtitle": "Wpisz organizację i dane dostępu.",
      "login.organization": "Organizacja",
      "login.organizationPlaceholder": "moja-firma",
      "login.email": "Email",
      "login.password": "Hasło",
      "login.hidePassword": "Ukryj hasło",
      "login.showPassword": "Pokaż hasło",
      "login.submit": "Zaloguj się",
      "login.submitting": "Logowanie...",
      "login.noAccount": "Nie masz konta?",
      "login.register": "Zarejestruj się",
      "login.securityNote": "Dostęp tenant-scoped, gotowy na 2FA i bezpieczną sesję.",
      "twoFa.title": "Weryfikacja 2FA",
      "twoFa.subtitle": "Kod z aplikacji uwierzytelniającej",
      "twoFa.codeLabel": "Kod weryfikacyjny",
      "twoFa.codeHint": "Wpisz 6-cyfrowy kod z aplikacji uwierzytelniającej",
      "twoFa.verify": "Zweryfikuj",
      "twoFa.verifying": "Weryfikacja...",
      "twoFa.backToLogin": "Wróć do logowania",
      "validation.orgRequired": "Slug organizacji jest wymagany",
      "validation.emailInvalid": "Nieprawidłowy adres email",
      "validation.passwordRequired": "Hasło jest wymagane",
    };
    return messages[key] ?? key;
  },
}));

vi.mock("@/hooks/use-auth", () => ({
  useAuth: () => ({
    login: loginMock,
    verify2FALogin: verify2FALoginMock,
  }),
}));

vi.mock("@/hooks/use-public-config", () => ({
  usePublicConfig: () => ({
    registration_mode: "invite",
    license_enabled: true,
  }),
}));

vi.mock("sonner", () => ({
  toast: { error: vi.fn() },
}));

describe("LoginPage Constellation Access", () => {
  beforeEach(() => {
    loginMock.mockReset();
    verify2FALoginMock.mockReset();
    loginMock.mockResolvedValue({ requires2FA: false });
  });

  it("renders the accepted OpenOMS login surface without non-functional extras", () => {
    render(<LoginPage />);

    expect(screen.getAllByText("OpenOMS")[0]).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Zaloguj się do panelu" })).toBeInTheDocument();
    expect(screen.getByText("Wpisz organizację i dane dostępu.")).toBeInTheDocument();
    expect(screen.getByLabelText(/^Organizacja/)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Email/)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Hasło/)).toHaveAttribute("type", "password");
    expect(screen.getByRole("button", { name: "Zaloguj się" })).toBeInTheDocument();
    expect(screen.getByText("Dostęp tenant-scoped, gotowy na 2FA i bezpieczną sesję.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Zarejestruj się" })).toHaveAttribute("href", "/register");
    expect(screen.queryByText(/PL · Polska/)).not.toBeInTheDocument();
  });

  it("keeps the password visibility toggle behavior", async () => {
    const user = userEvent.setup();
    render(<LoginPage />);

    const password = screen.getByLabelText(/^Hasło/);
    await user.click(screen.getByRole("button", { name: "Pokaż hasło" }));
    expect(password).toHaveAttribute("type", "text");
    await user.click(screen.getByRole("button", { name: "Ukryj hasło" }));
    expect(password).toHaveAttribute("type", "password");
  });

  it("renders the 2FA card in the same visual shell after login requires a code", async () => {
    const user = userEvent.setup();
    loginMock.mockResolvedValue({ requires2FA: true, tempToken: "temp-token" });

    render(<LoginPage />);
    await user.type(screen.getByLabelText(/^Organizacja/), "openoms");
    await user.type(screen.getByLabelText(/^Email/), "rafal@openoms.org");
    await user.type(screen.getByLabelText(/^Hasło/), "Password123");
    await user.click(screen.getByRole("button", { name: "Zaloguj się" }));

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "Weryfikacja 2FA" })).toBeInTheDocument();
    });
    expect(screen.getByLabelText("Kod weryfikacyjny")).toHaveFocus();
    expect(screen.getByText("Dostęp tenant-scoped, gotowy na 2FA i bezpieczną sesję.")).toBeInTheDocument();
  });
});
