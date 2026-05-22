import { render, screen, within } from "@testing-library/react";

import HelpPage from "./page";

const translations: Record<string, string> = {
  faq: "FAQ",
  faqDescription: "Odpowiedzi na najczestsze pytania sprzedawcow.",
  officialSupportCta: "Napisz do supportu",
  officialSupportDescription:
    "Napisz do nas, jesli potrzebujesz pomocy z kontem, integracja albo procesem obslugi zamowien.",
  officialSupportMeta: "Odpowiadamy w kanale obslugi OpenOMS.",
  officialSupportTitle: "Pomoc OpenOMS",
  ossDescription: "Dla problemow technicznych z publicznym repo uzyj GitHub Issues.",
  ossTitle: "Open-source i self-hosting",
  prepareContext: "Modul, integracja albo numer zamowienia",
  prepareEvidence: "Krotki opis, screenshot albo komunikat bledu",
  prepareTenant: "Nazwa firmy lub organizacji",
  prepareTitle: "Co dolaczyc do zgloszenia",
  reportBug: "GitHub Issues",
  subtitle: "Skontaktuj sie z OpenOMS albo sprawdz sprawdzone materialy.",
  title: "Pomoc",
  userGuide: "Poradnik uzytkownika",
  userGuideDescription: "Najwazniejsze kroki konfiguracji i pracy operacyjnej.",
};

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => translations[key] ?? key,
}));

describe("HelpPage", () => {
  it("prioritizes official customer support over community issue reporting", () => {
    render(<HelpPage />);

    const supportLink = screen.getByRole("link", { name: /napisz do supportu/i });
    expect(supportLink).toHaveAttribute("href", "mailto:support@openoms.org");

    expect(screen.getByText("Pomoc OpenOMS")).toBeInTheDocument();
    expect(screen.getByText("Co dolaczyc do zgloszenia")).toBeInTheDocument();
    expect(screen.queryByText(/discord/i)).not.toBeInTheDocument();

    const ossSection = screen.getByText("Open-source i self-hosting").closest("section");

    expect(ossSection).not.toBeNull();
    expect(
      within(ossSection as HTMLElement).getByRole("link", { name: /github issues/i })
    ).toHaveAttribute("href", "https://github.com/openoms-org/openoms/issues");
  });
});
