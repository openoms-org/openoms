import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const dashboardSourceRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../..");

const auditedFiles = [
  "components/dashboard/stat-cards.tsx",
  "app/(auth)/register/page.tsx",
  "app/(dashboard)/pick-pack/page.tsx",
  "app/(public)/supplier-portal/page.tsx",
  "app/(dashboard)/marketplaces/allegro/delivery/page.tsx",
  "app/(dashboard)/marketplaces/allegro/disputes/page.tsx",
  "app/(dashboard)/marketplaces/allegro/ratings/page.tsx",
  "app/(dashboard)/marketplaces/allegro/policies/page.tsx",
  "app/(dashboard)/marketplaces/allegro/catalog/page.tsx",
  "app/(dashboard)/marketplaces/allegro/finance/page.tsx",
  "app/(dashboard)/marketplaces/allegro/promotions/page.tsx",
  "app/(dashboard)/workflows/[id]/page.tsx",
  "components/workflow/workflow-toolbar.tsx",
  "components/workflow/node-config-panel.tsx",
  "app/(dashboard)/error.tsx",
  "app/(public)/error.tsx",
  "components/editor/description-editor.tsx",
  "app/(dashboard)/products/[id]/listings/page.tsx",
];

const forbiddenPhrases = [
  "Nie udalo sie",
  "Brak zamowien",
  "Brak dostępnych",
  "Brak dostepnych",
  "Wybierz plan",
  "Miesiecznie",
  "Rocznie",
  "Najpopularniejszy",
  "Kompletowanie",
  "Pakowanie",
  "Zakonczone",
  "Anulowane",
  "Utworz",
  "Sprobuj ponownie",
  "Portal Dostawcy",
  "Zamowienia zakupowe",
  "Wiadomosci",
  "Potwierdz zamowienie",
  "Oznacz jako wyslane",
  "Zapisz",
  "Wybierz zdarzenie",
  "Wybierz typ akcji",
  "Cos poszlo nie tak",
  "Naglowek 1",
  "Generuj opis",
  "Popraw opis",
  "Przetlumacz na polski",
  "Brak ofert marketplace",
  "Wybierz cennik wysylki",
  "Utworz domyslna polityke",
  "Polityka wysylki",
  "Stan magazynowy",
];

describe("dashboard audited copy", () => {
  it("does not keep audited Polish UI copy hardcoded in source files", () => {
    const violations = auditedFiles.flatMap((file) => {
      const source = readFileSync(resolve(dashboardSourceRoot, file), "utf8");
      return forbiddenPhrases
        .filter((phrase) => source.includes(phrase))
        .map((phrase) => `${file}: ${phrase}`);
    });

    expect(violations).toEqual([]);
  });
});
