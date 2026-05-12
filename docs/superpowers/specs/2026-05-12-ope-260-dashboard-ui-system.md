# OPE-260: OpenOMS dashboard UI system

## Cel

Ujednolicic caly dashboard OpenOMS pod kierunek wypracowany w OPE-242: **operations control tower** dla orkiestracji e-commerce. Produkt ma wygladac i dzialac jak jedna spojna aplikacja dla operatorow i adminow, nie jak zbior niezaleznych ekranow.

To jest spec dla calego UI systemu. Implementacja bedzie podzielona na mniejsze PR-y, ale wszystkie PR-y maja wynikac z tego samego jezyka wizualnego, komponentow i zasad interakcji.

## Definicja produktu

OpenOMS nie jest sklepem internetowym ani marketplace managerem. OpenOMS jest warstwa orkiestracji operacji e-commerce:

- przyjmuje zamowienia z kanalow sprzedazy,
- normalizuje i waliduje dane,
- kieruje realizacje przez magazyn i logistyke,
- synchronizuje stany, oferty, dokumenty i zdarzenia,
- pokazuje wyjatki, blokady i miejsca wymagajace reakcji,
- daje adminowi narzedzia do konfiguracji sprawdzonych integracji i procesow.

UI ma wzmacniac ten model. Ekrany nie powinny wygladac jak storefront, CRM marketingowy albo demo funkcji. Maja byc spokojnym, gestym, profesjonalnym narzedziem do codziennej pracy.

## Zakres

OPE-260 obejmuje caly dashboard:

- ekrany operatorskie: zamowienia, przesylki, produkty, zwroty, klienci, pakowanie, pick-pack,
- ekrany admina: ustawienia, uzytkownicy, role, firma, bezpieczenstwo, integracje, logistyka,
- obszary kanalow i providerow: marketplaces, carriers, suppliers, feeds, sync,
- obszary diagnostyczne: audit, webhooki, sync jobs, raporty,
- auth, onboarding i publiczne flow, jezeli dotykaja pierwszego doswiadczenia klienta lub operatora.

Zakres nie oznacza jednego wielkiego PR-a. Zakres oznacza jeden docelowy standard. Migracja ma isc falami, z review i testami po kazdym kroku.

## Zasady nadrzedne

1. **Jeden system, wiele ekranow**  
   Kazdy ekran ma skladac sie z tych samych prymitywow: page header, sekcje, powierzchnie robocze, tabele, formularze, statusy, action bary i empty states.

2. **Najpierw komponenty, potem migracje**  
   Nie poprawiamy kazdego ekranu ad hoc. Najpierw tworzymy stabilny zestaw komponentow, potem przepisujemy ekrany na ten zestaw.

3. **Nie pokazujemy niedzialajacych interakcji**  
   Jezeli nie ma backendu, sprawdzonego flow, route albo uprawnienia, UI nie pokazuje aktywnej akcji. Dopuszczalna jest pasywna informacja statusowa, ale nie przycisk sugerujacy gotowa funkcje.

4. **Client-ready mode jest twarda bramka**  
   Providerzy, moduly i opcje niesprawdzone dla klienta pozostaja ukryte albo pasywne. InPost moze byc gotowy; reszta providerow wymaga readiness matrix przed udostepnieniem jako realny wybor.

5. **Admin i user maja ten sam jezyk UI**  
   Admin widzi wiecej konfiguracji, ale nie inny produkt. Ustawienia maja byc spokojne, logiczne i tak samo dopracowane jak ekrany operatorskie.

6. **Gestosc bez chaosu**  
   OpenOMS jest narzedziem operacyjnym. UI moze byc gesty informacyjnie, ale musi miec wyrazna hierarchie, przewidywalne akcje i czytelne stany.

## Visual system

Kierunek wizualny pozostaje zgodny z OPE-242:

- neutralne off-white tlo aplikacji,
- biale powierzchnie robocze,
- cienkie neutralne ramki,
- grafitowy tekst,
- spokojny teal/blue dla aktywnych elementow,
- zielony, amber i czerwony tylko dla znaczenia statusowego,
- radius 6-8px,
- delikatne cienie tylko dla elementow warstwowych, np. modal, dropdown, command palette,
- bez gradientowych blobow, hero sekcji, dekoracyjnego glassmorphism i marketingowego tonu.

Typografia:

- kompaktowa skala tekstu,
- naglowki ekranow nie moga dominowac nad praca,
- liczby i metryki uzywaja tabular figures,
- etykiety i helper text sa czytelne, ale nie glosniejsze niz dane.

Gestosc:

- listy i tabele maja byc latwe do skanowania,
- filtry i akcje sa blisko danych, ktorych dotycza,
- secondary metadata jest sciszona wizualnie,
- ekran nie powinien wymagac scrolla tylko dlatego, ze layout uzywa za duzych kart.

## Component architecture

### AppShell

Istniejacy shell z OPE-242 zostaje baza:

- sidebar z grupami domenowymi,
- topbar z command palette, statusem polaczenia i user menu,
- jasny, spokojny canvas roboczy,
- responsive mobile nav.

Prace OPE-260 nie powinny przebudowywac shella od zera. Moga go dopolerowac, jezeli migracja ekranow pokaze powtarzalne problemy.

### PageHeader

Kazdy ekran dashboardu powinien zaczynac sie od jednego wspolnego naglowka:

- tytul,
- krotki opis funkcji lub kontekstu,
- opcjonalne breadcrumbs,
- opcjonalny primary action,
- opcjonalne secondary actions w menu lub action barze.

Primary action musi istniec tylko wtedy, gdy akcja dziala. Nie dodajemy disabled "coming soon" jako stalego wzoru.

### PageSection

Sekcja grupuje logiczny fragment pracy:

- tytul sekcji,
- pomocniczy opis,
- optional actions,
- dzieci: tabela, formularz, lista, status panel albo custom content.

Sekcje nie sa dekoracyjnymi kartami w kartach. Sa unframed albo maja jedna powierzchnie robocza, zalezne od typu ekranu.

### Surface

Surface to podstawowa biala powierzchnia robocza:

- border 1px,
- radius 6-8px,
- kontrolowane paddingi,
- bez glebokich cieni,
- warianty: default, muted, warning, danger, success.

Surface sluzy do list, formularzy, detail blocks i settings panels. Nie powinien zastapic semantycznych komponentow, jezeli ekran ma lepszy wzorzec.

### DataView

DataView standaryzuje tabele i listy:

- toolbar z filtrami, wyszukiwaniem, density i akcjami,
- table/list body,
- loading skeleton,
- empty state,
- error state,
- pagination,
- row actions,
- optional bulk actions.

Tabele musza miec przewidywalny rytm:

- lewa kolumna: glowny identyfikator lub nazwa,
- srodek: statusy i najwazniejsze pola,
- prawa strona: daty, kwoty, akcje,
- akcje destrukcyjne schowane za confirm dialog.

### FormLayout

Formularze powinny miec wspolny uklad:

- widoczne labelki,
- helper text pod polem, jezeli pole wymaga wyjasnienia,
- blad przy polu,
- sekcje tematyczne,
- action bar na dole lub w headerze, zalezne od dlugosci formularza,
- spojnosc kolejnosci przyciskow: primary po prawej, cancel/back jako secondary.

Nie uzywamy placeholdera jako jedynej etykiety. Nie pokazujemy pol providerow, ktore nie dotycza wybranego typu.

### DetailLayout

Ekrany szczegolow powinny miec:

- summary header z kluczowym statusem,
- glowna kolumne z danymi i timeline,
- boczny panel z metadanymi, akcjami i diagnostyka,
- jasna separacje miedzy informacja a akcja.

Przyklad: order detail, shipment detail, product detail, integration detail.

### SettingsLayout

Ustawienia powinny miec osobny, spokojny wzorzec:

- lista kategorii albo lokalne tabs,
- pojedynczy temat na sekcje,
- zapisywanie zmian z widocznym feedbackiem,
- ostrzezenia przy zmianach ryzykownych,
- brak mieszania ustawien operacyjnych i developerskich na jednym ekranie.

### Status primitives

Statusy powinny byc semantyczne, nie surowo providerowe:

- `ok`,
- `warning`,
- `problem`,
- `inactive`,
- `draft`,
- `pending`,
- `blocked`.

Kolor nie moze byc jedynym nosnikiem znaczenia. Badge powinien miec tekst, a wazne stany moga miec ikone.

### EmptyState

EmptyState musi prowadzic do realnej pracy:

- jezeli akcja jest gotowa, pokazuje jedno CTA,
- jezeli funkcja wymaga konfiguracji, prowadzi do istniejacego ekranu konfiguracji,
- jezeli funkcja nie jest client-ready, nie obiecuje jej aktywnym przyciskiem.

EmptyState nie moze byc marketingowa reklama funkcji.

## Screen families

### Operator lists

Dotyczy: orders, shipments, products, returns, customers, packing, pick-pack.

Wzorzec:

- PageHeader,
- DataView toolbar,
- tabela/lista z gestymi danymi,
- row action menu,
- bulk actions tylko tam, gdzie backend je obsluguje,
- empty/loading/error states z tego samego systemu.

### Operational details

Dotyczy: order detail, shipment detail, product detail, return detail, customer detail.

Wzorzec:

- DetailLayout,
- status i najwazniejszy identyfikator w headerze,
- timeline lub activity tylko tam, gdzie mamy dane,
- panel boczny dla metadanych i dostepnych akcji,
- akcje statusowe przez potwierdzone flow.

### Provider setup

Dotyczy: marketplaces, integrations, carriers, suppliers.

Wzorzec:

- provider picker pokazuje tylko providerow dopuszczonych przez readiness,
- zewnetrzne uslugi maja jeden wspolny provider identity: logo/wordmark, nazwe, kategorie i fallback dla nieznanych providerow,
- logo/wordmark stosujemy dla kurierow, marketplace'ow, fakturowania i dostawcow; moduly systemowe nadal uzywaja spojnych ikon funkcjonalnych,
- niegotowi providerzy sa ukryci w client-ready mode,
- formularz providera pokazuje tylko pola wymagane przez ten provider,
- test connection jest widoczny tylko jesli flow istnieje,
- status integracji jest czytelny i linkuje do diagnostyki, jezeli ta istnieje.

### Admin settings

Dotyczy: company, users, roles, security, billing, accounting, webhooks, sync, inventory, price lists, print templates.

Wzorzec:

- SettingsLayout,
- kategorie zamiast jednej dlugiej sciany kart,
- ostrozne primary actions,
- potwierdzenia dla zmian ryzykownych,
- jasne stany zapisane/niezapisane.

### Auth, onboarding, public flows

Dotyczy: login, register, onboarding, return request, supplier portal.

Wzorzec:

- ten sam brand i typografia,
- mniej gesty layout niz dashboard,
- widoczne labelki i feedback,
- minimalne zaufane informacje, bez obietnic niedzialajacych modulow,
- mobile-first.

## Navigation model

Nawigacja ma odzwierciedlac orkiestracje, nie liste implementacji.

Docelowe obszary:

- Pulpit operacyjny,
- Sprzedaz i zamowienia,
- Katalog,
- Logistyka,
- Kanaly sprzedazy,
- Zakupy i dostawcy,
- Raporty i diagnostyka,
- Automatyzacje i narzedzia,
- Ustawienia,
- Pomoc.

Zmiany nawigacji musza byc ostrozne:

- nie chowamy gotowego, potrzebnego ekranu bez alternatywnej sciezki,
- nie pokazujemy ukrytych route przez command palette,
- admin-only pozostaje admin-only,
- route guard i nav musza korzystac z tych samych zasad readiness.

## Readiness and feature visibility

Kazda widoczna funkcja powinna miec jeden z ponizszych statusow:

- `client_ready`: uzytkownik moze jej uzywac,
- `admin_ready`: admin moze ja skonfigurowac lub testowac,
- `diagnostic_only`: widoczna jako informacja lub log, bez aktywnych akcji,
- `hidden`: nie pokazujemy w client-ready mode.

Ta zasada dotyczy:

- providerow marketplace,
- kurierow,
- integracji fakturowania,
- automatyzacji,
- raportow,
- narzedzi eksperymentalnych.

Provider lub modul bez potwierdzonego dzialania nie moze byc aktywnym wyborem dla klienta.

## Error, loading and feedback

Kazdy komponent danych powinien miec spojnosc stanow:

- loading: skeleton dopasowany do docelowego layoutu,
- empty: jednoznaczne wyjasnienie i realne CTA,
- error: lokalny komunikat, retry tylko jesli refetch dziala,
- success: subtelny toast lub inline feedback,
- saving: disabled primary action z progressem,
- destructive action: confirm dialog.

Bledy autoryzacji wynikajace z ukrytych admin-only requestow nie powinny byc widoczne dla zwyklego uzytkownika. Hooki i komponenty maja respektowac role i readiness przed wykonaniem requestow.

## Accessibility and responsive rules

Minimalny standard:

- kazdy icon-only button ma `aria-label`,
- focus ring pozostaje widoczny,
- tab order odpowiada kolejnosci wizualnej,
- targety klikalne min. 44px tam, gdzie sa glowne akcje,
- tekst nie wychodzi poza kontenery na mobile,
- zadnych hover-only akcji jako jedynej sciezki,
- kontrast tekstu i statusow ma byc czytelny,
- reduced motion respektowany dla animacji.

Responsive:

- desktop jest glownym trybem pracy operacyjnej,
- mobile musi byc uzywalne dla podstawowych flow,
- tabele moga przechodzic w listy lub poziomy scroll tylko tam, gdzie nie tracimy dostepnosci,
- formularze na mobile sa jednokolumnowe.

## Implementation waves

### PR 1: UI foundations

Cel:

- doprecyzowac shared primitives,
- uporzadkowac PageHeader, EmptyState, StatusBadge, DataTable,
- dodac PageSection, Surface, ActionBar, FormLayout, DetailLayout i SettingsLayout, jezeli lokalny kod potwierdzi potrzebe,
- nie migrowac jeszcze wszystkiego naraz.

Powiazane issue: OPE-244 jako pierwszy slice implementacyjny.

### PR 2: operator lists

Cel:

- orders,
- shipments,
- products,
- returns,
- customers,
- packing / pick-pack, jezeli sa client-ready.

### PR 3: provider and logistics surfaces

Cel:

- integrations,
- marketplaces,
- carriers,
- suppliers,
- stock sync,
- provider readiness visibility.

### PR 4: admin and settings

Cel:

- settings layout,
- users / roles / security,
- company / billing / accounting,
- webhooks / sync jobs / inventory / templates.

### PR 5: auth, onboarding and public flows

Cel:

- login/register,
- onboarding,
- return request,
- supplier portal,
- consistent trust and feedback states.

### PR 6: final polish and QA

Cel:

- visual QA,
- accessibility pass,
- responsive pass,
- command palette/nav consistency,
- remove dead links and hidden feature leaks,
- update docs.

## Testing strategy

Per PR:

- targeted unit/component tests for changed shared components,
- existing page tests updated when behavior changes,
- `npm run lint`,
- targeted `vitest`,
- `next build`,
- browser QA for desktop and mobile widths.

Before PR ready:

- no active action without working route/API,
- no hidden feature exposed through nav or command palette,
- common states checked: loading, empty, error, success,
- screenshots or manual notes for major migrated screens.

Before public push:

- full `./scripts/local-ci.sh` from public repo, per project rules.

## Acceptance criteria

OPE-260 is successful when:

- dashboard, admin, auth and public flows share one visual language,
- OPE-242 dashboard no longer feels like an isolated island,
- tables, forms, details, settings and empty states use shared primitives,
- role and readiness rules are consistent across nav, command palette and page content,
- unready functionality is not usable by customers,
- the app remains fast, accessible and readable on desktop and mobile,
- future features can be added by composing existing UI primitives instead of inventing a new page style.

## Non-goals

This spec does not certify new providers, rewrite backend flows, add missing endpoint behavior, or make hidden features client-ready. It only defines how the UI should expose and compose functionality that is real, checked and allowed by readiness rules.

Hardening work such as Docker secret handling remains important, but it is intentionally a separate track after the UI system direction is captured.

## Status

2026-05-12: OPE-244 began the first implementation slice for the UI system. The slice adds shared layout primitives and backward-compatible upgrades to PageHeader, EmptyState, StatusBadge and DataTable, so future route migrations can use one visual system instead of per-page styling.

2026-05-12: OPE-261/OPE-262 continue the first polish wave after production visual QA. The slice refreshes the default dashboard CTA treatment, lets PageHeader and EmptyState actions use explicit button variants, moves visible shipment/provider setup forms toward FormSection/FormActions, and fixes the missing marketplace breadcrumb translation seen on production.

2026-05-12: OPE-264 adds the provider identity layer for the UI system. Carrier, marketplace, invoicing and generic integration surfaces should use one shared ProviderLogo/ProviderInfo source of truth, with brand-colored wordmarks now and a clean path for official logo assets later.
