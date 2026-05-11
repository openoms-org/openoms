# OPE-241: Client-ready dashboard surface

## Cel

Przygotować dashboard OpenOMS do pierwszego użycia przez klientów przez ograniczenie widocznej powierzchni produktu do sprawdzonych, zrozumiałych ścieżek. To nie jest pełny redesign. To szybki, produktowy cleanup przed rolloutem: mniej opcji, mniej technicznego szumu, zero placeholderów i brak widocznych funkcji, które wyglądają jak niedokończone.

## Problem

Obecny dashboard pokazuje bardzo szeroką nawigację, około 55 pozycji po rozwinięciu grup. Dla rozwijanego systemu OMS ma to sens architektonicznie, ale dla pierwszego klienta tworzy zbyt duży ciężar poznawczy. Użytkownik widzi jednocześnie podstawowe moduły, moduły eksperymentalne, moduły administracyjne oraz funkcje wymagające jeszcze pełnej walidacji end-to-end.

W audycie produkcyjnego UI znaleziono też widoczne problemy z zaufaniem:

- placeholdery typu `Tytuł` i `Podtytuł` na ekranach dostawców i dropshippingu,
- surowe klucze tłumaczeń na ekranie dostarczeń webhooków,
- robocze lub błędne copy na ekranach SMS i bezpieczeństwa,
- widoczne moduły zaawansowane bez danych lub z komunikatem sugerującym stan beta,
- wpis marketplace w stanie `error`, widoczny od razu na ekranie integracji.

## Wybrany kierunek

Wybrany kierunek to **client-ready mode**: domyślna produkcyjna powierzchnia dashboardu pokazuje tylko moduły, które są sensowne dla pierwszego klienta i nie wyglądają jak funkcje w budowie. Reszta zostaje w kodzie, ale jest ukryta w nawigacji i palecie poleceń przez jawne metadane gotowości.

Alternatywy odrzucone:

- **Pełny redesign dashboardu**: za duży zakres na teraz; ryzyko spowolnienia rolloutów.
- **Tylko poprawa tekstów**: usuwa część problemów, ale nie rozwiązuje przeciążonej nawigacji.
- **Oznaczanie niedokończonych modułów jako beta**: lepsze dla produktu dojrzałego, ale przed pierwszym klientem nadal osłabia zaufanie.

## Zakres pierwszego PR

Pierwszy PR obejmuje publiczny dashboard:

- uproszczenie widocznej nawigacji dla produkcyjnego klienta,
- ukrycie niedokończonych, zaawansowanych lub technicznych modułów z menu i command palette,
- naprawę najbardziej widocznych placeholderów i błędnych tłumaczeń,
- poprawę pustych stanów na kluczowych ekranach,
- lekki polish wizualny: spójniejsza hierarchia, statusy, spacing i czytelność.

Bez zmian w backendzie, chyba że frontend ujawni problem, którego nie da się poprawić bez małego, bezpiecznego dopasowania API.

## Poza zakresem

Ten PR nie obejmuje:

- pełnego redesignu dashboard shell,
- nowego systemu brandingu,
- przebudowy wszystkich tabel i formularzy,
- wdrażania nowych funkcji,
- usuwania modułów z kodu lub API,
- zmian w publicznym Helm chartcie albo enterprise deploy pipeline.

## Nawigacja klienta

Domyślna powierzchnia klienta powinna mieć krótką, przewidywalną strukturę:

- Pulpit
- Sprzedaż: Zamówienia, Klienci, Zwroty
- Katalog: Produkty, Kategorie
- Operacje: Przesyłki, Marketplace
- Administracja: Firma, Użytkownicy, Role, Bezpieczeństwo
- Pomoc

Do ukrycia w pierwszym client-ready PR, dopóki nie zostaną osobno zweryfikowane:

- Fakturowanie i faktury, jeśli nie mamy potwierdzonego flow klienta,
- szablony druku,
- kurierzy, pakowanie, Pick & Pack, magazyny, inwentaryzacja, dokumenty magazynowe, stock sync,
- feedy produktowe i synchronizacja ofert,
- raporty zaawansowane: forecast, carbon, VAT OSS, reconciliation, repricing,
- zaopatrzenie: dostawcy, purchase orders, dropshipping,
- automatyzacja i workflow builder,
- importy, usuwanie tła, waluty, subskrypcje zamówień, loyalty, segmenty klientów,
- billing/subscription w aplikacji klienta, jeśli nie obsługujemy tego jako customer-facing flow,
- cenniki, księgowość, VAT OSS settings, inventory control, SMS, webhook deliveries, sync jobs, audit log.

Moduły ukryte nie powinny znikać z kodu. Powinny dostać jawny status gotowości w definicji nawigacji, aby później można było je włączać kontrolowanie.

## Model gotowości funkcji

`navItems` powinno rozróżniać przynajmniej:

- `ready`: widoczne dla klienta,
- `internal`: dostępne tylko dla owner/admin/dev lub przez bezpośredni URL, ale nie eksponowane w menu,
- `beta`: ukryte domyślnie w produkcji, możliwe do włączenia później flagą,
- `hidden`: nie pokazywać w nawigacji ani command palette.

Pierwszy PR może użyć prostszego wariantu technicznego, jeśli będzie pasował do obecnej architektury, ale decyzja musi zostać zapisana w kodzie czytelnie i bez magicznych list rozproszonych po komponentach.

## Gotowość providerów i akcji

Client-ready surface nie może kończyć się na poziomie ekranu. Jeśli klient widzi moduł, to akcje i providery dostępne w tym module też muszą być gotowe do użycia. Dotyczy to szczególnie:

- kurierów,
- marketplace,
- hurtowni i dostawców,
- integracji księgowych,
- płatności i billing,
- automatyzacji,
- importów i eksportów.

Zasada produktu: klient nie powinien móc wybrać providera albo uruchomić akcji, której nie zweryfikowaliśmy jako działającej end-to-end. Provider niegotowy ma być ukryty albo niedostępny w client-ready mode. Samo oznaczenie `beta` nie wystarcza dla pierwszego klienta, jeśli funkcja może popsuć jego workflow.

Pierwszy PR powinien co najmniej uniknąć eksponowania niezweryfikowanych providerów w widocznych ścieżkach klienta. Pełna macierz certyfikacji providerów jest osobnym follow-upem, bo wymaga testów sandbox/produkcyjnych i decyzji biznesowych dla każdej integracji.

Docelowy model powinien mieć jedno źródło prawdy dla gotowości capability/providerów, np. status:

- `ready`: widoczne i używalne dla klienta,
- `internal`: dostępne tylko operacyjnie/dev,
- `beta`: możliwe do włączenia później flagą, ale niewidoczne domyślnie,
- `disabled`: niedostępne w UI.

W UI status `ready` powinien być jedynym statusem, który pozwala klientowi użyć providera w normalnym flow.

## Copy i zaufanie

Widoczne placeholdery muszą zniknąć przed rolloutem:

- `/suppliers`: zamienić `Tytuł` na rzeczywisty nagłówek lub ukryć ekran.
- `/dropship-orders`: zamienić `Tytuł` i `Podtytuł` albo ukryć ekran.
- `/settings/webhooks/deliveries`: naprawić surowe klucze `settings.webhooks.*` albo ukryć ekran.
- `/settings/sms`: usunąć roboczy tekst, przykładowe dane wyglądające jak prawdziwa konfiguracja i błędne labelki typu `Hasło DHL`.
- `/settings/security`: poprawić tekst instrukcji 2FA i posklejane fragmenty.
- `/help`: dla klienta SaaS pomoc nie powinna prowadzić głównie do GitHuba/Discorda. Minimum: neutralny ekran wsparcia z kontaktem i dokumentacją użytkownika.

## Empty states

Pusty stan powinien odpowiadać na trzy pytania:

- co tu będzie,
- dlaczego teraz jest pusto,
- jaka jest jedna najlepsza następna akcja.

Nie pokazujemy długich tabel z paginacją, jeśli ekran nie ma żadnych danych i jedyną sensowną akcją jest konfiguracja integracji albo dodanie pierwszego rekordu. Dla pierwszego klienta puste stany powinny prowadzić do konfiguracji firmy, połączenia marketplace albo dodania/importu produktu.

## Visual polish

Styl pozostaje spokojny, użytkowy i SaaS-owy. Nie robimy marketingowej przebudowy ani ciężkich efektów wizualnych.

Zasady:

- mniej ekspozycji funkcji, więcej jasnej hierarchii,
- jeden główny CTA na ekran,
- statusy zawsze czytelne tekstowo, nie tylko kolorem,
- spacing i typografia zgodne z obecnym shadcn/Tailwind stylem,
- menu ma być skanowalne, bez długiej listy technicznych modułów,
- empty states powinny wyglądać jak świadomy stan produktu, nie jak brak implementacji.

## Test plan

Minimum dla PR:

- test jednostkowy lub component test dla filtrowania nawigacji i command palette,
- test albo manual pass, że widoczne selektory providerów nie pokazują pozycji niegotowych w client-ready mode,
- `npm run lint` w dashboardzie,
- testy zmienionych komponentów,
- Playwright/manual browser pass po produkcyjnych ścieżkach UI: `/`, `/orders`, `/products`, `/customers`, `/returns`, `/settings/company`, `/settings/security`, `/help`,
- sprawdzenie, że ukryte moduły nie są widoczne w sidebarze ani command palette,
- `./scripts/local-ci.sh` przed pushem do public repo.

## Akceptacja

PR jest gotowy, gdy:

- klient nie widzi niedokończonych lub technicznych modułów w domyślnej nawigacji,
- command palette nie odkrywa ukrytych modułów,
- klient nie może wybrać niezweryfikowanego providera w widocznych flow,
- najgorsze placeholdery i surowe klucze tłumaczeń są usunięte albo ukryte razem z modułem,
- podstawowe ekrany mają zrozumiałe puste stany,
- dashboard nadal działa w aktualnym trybie owner/admin,
- zmiany nie naruszają public/enterprise boundary.
