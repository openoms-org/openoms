# OPE-245 Customer Help Design

## Goal

Usprawnic strone Pomoc tak, zeby klient SaaS widzial jasna sciezke kontaktu z OpenOMS, a linki open-source/community byly dostepne jako kontekst dodatkowy, nie jako domyslny kanal wsparcia.

## Scope

- Dotyczy tylko dashboardu w public repo.
- Nie wlaczamy ukrytego modulu helpdesk/Freshdesk, poniewaz readiness oznacza go jako zablokowany.
- Nie dodajemy backendu ani runtime configuration w tym kroku.
- Nie zmieniamy nawigacji: `/help` zostaje widoczna i gotowa.

## Current Problem

Obecna strona Pomoc prowadzi najpierw do GitHub docs, Discorda i GitHub Issues. To jest sensowne dla open-source, ale dla klienta SaaS wyglada jak brak oficjalnego wsparcia operacyjnego. Bezposredni kontakt istnieje tylko jako mala stopka.

## Proposed Experience

Strona bedzie miala trzy poziomy informacji:

1. **Primary support**: glowny panel kontaktu z OpenOMS z jednym czytelnym CTA mailowym.
2. **Self-service**: poradnik uzytkownika i FAQ jako szybkie linki do sprawdzonych materialow.
3. **Open-source / self-hosted**: osobna, spokojniejsza sekcja z GitHub Issues dla problemow technicznych zwiazanych z publicznym repo.

## Content Rules

- Copy ma byc proste, klientocentryczne i nietechniczne.
- GitHub nie moze byc domyslna sciezka klienta SaaS.
- Jezeli link lub funkcja nie jest pewna, nie pokazujemy jej.
- Nie pokazujemy Discorda w tym kroku, bo nie mamy potwierdzonego support procesu dla tego kanalu.
- Kontakt docelowy w tym kroku to `support@openoms.org`.

## Public vs Enterprise Decision

Ta zmiana zostaje w public repo, bo dashboard jest wspolna baza dla OpenOMS. Nie dodajemy enterprise-only konfiguracji, poniewaz obecny dashboard nie ma bezpiecznego runtime public-config mechanizmu dla tego typu tresci po buildzie obrazu. Uzywamy public-safe domyslnego kontaktu `support@openoms.org`. Jesli pozniej bedziemy chcieli per-tenant albo enterprise-only routing supportu, powinien powstac osobny endpoint public config lub tenant settings, zamiast build-time `NEXT_PUBLIC_*`.

## UI Structure

- Header: `Pomoc` / `Help` z opisem, ze to miejsce kontaktu i materialow.
- Main card: oficjalna pomoc OpenOMS, ikona supportu, mail CTA, dodatkowy opis czego dotyczy kontakt.
- Preparation card: krotka lista informacji, ktore warto dolaczyc do zgloszenia.
- Resource cards: poradnik i FAQ.
- OSS card: mniej dominujaca sekcja dla self-hosted/open-source, z linkiem do GitHub Issues.

## Accessibility and UX

- Jeden primary CTA na ekran.
- Linki zewnetrzne otwieraja sie w nowej karcie i maja `rel="noopener noreferrer"`.
- Ikony sa dekoracyjne tam, gdzie tekst juz opisuje cel.
- Layout mobile-first, bez gestych zagniezdzonych kart.

## Tests

- Dodac test renderowania strony Help z mockiem `next-intl`.
- Test powinien potwierdzac:
  - oficjalny kontakt supportu jest glowna akcja,
  - GitHub Issues pozostaje dostepny,
  - Discord nie jest renderowany,
  - mailto uzywa `support@openoms.org`.

## Documentation

- Zaktualizowac `docs/audit/feature-readiness-matrix-2026-05-11.md`, zeby opis `/help` mowil o klientocentrycznej stronie wsparcia i braku niedzialajacych kanalow.
