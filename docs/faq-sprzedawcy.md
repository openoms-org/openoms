# OpenOMS — FAQ dla sprzedawcow

Najczesciej zadawane pytania przez sprzedawcow korzystajacych z OpenOMS.

---

## Q: Czy OpenOMS jest darmowy?

Tak. OpenOMS to oprogramowanie open-source na licencji AGPLv3. Mozesz korzystac z niego za darmo, hostujac na wlasnym serwerze lub korzystajac z Dockera. Nie ma ukrytych oplat, limitow zamowien ani obowiazkowych subskrypcji. Pakiety pomocnicze (SDK, biblioteki) sa na licencji MIT.

---

## Q: Jak dodac wielu uzytkownikow do jednej firmy?

1. Przejdz do **Uzytkownicy** w sekcji "Ogolne" w menu bocznym (wymaga roli Administratora lub Wlasciciela).
2. Kliknij **Dodaj uzytkownika**.
3. Podaj email, imie, nazwisko i wybierz role (Wlasciciel, Administrator lub Czlonek).

Mozesz tez tworzyc wlasne role z indywidualnie dobranymi uprawnieniami w **Role** (sekcja "Ogolne"). Na przyklad mozesz stworzyc role "Magazynier" z dostepem tylko do przesylek i inwentaryzacji.

---

## Q: Czy moge polaczyc kilka kont Allegro?

Tak. W sekcji **Kanaly sprzedazy** mozesz dodac wiele integracji Allegro. Kazda integracja laczy sie z osobnym kontem Allegro. Zamowienia ze wszystkich kont trafiaja na wspólna liste zamowien — mozesz je rozrozniac po zrodle.

Aby dodac kolejne konto:
1. Przejdz do **Polaczenia** w sekcji "Narzedzia".
2. Kliknij **Nowe polaczenie** i wybierz Allegro.
3. Przejdz przez proces autoryzacji OAuth2 dla drugiego konta.

---

## Q: Jak importowac produkty z CSV?

1. Przejdz do **Import produktow** w sekcji "Katalog" w menu bocznym.
2. Pobierz szablon CSV — zawiera przykladowe wiersze i opis kolumn (nazwa, SKU, cena, ilosc, opis itd.).
3. Uzupelnij plik swoimi danymi.
4. Wgraj plik na strone importu.
5. System przetworzy plik i pokaze raport: ile produktow zaimportowano, ile mialo bledy.

Jesli juz masz produkty w systemie i chcesz je zaktualizowac, uzyj tego samego importu — system dopasuje produkty po SKU.

Wiecej: [Poradnik uzytkownika — Produkty](poradnik-uzytkownika.md#6-produkty)

---

## Q: Jak ustawic wlasne statusy zamowien?

Domyslnie OpenOMS ma 14 statusow (Nowe, Potwierdzone, W realizacji itd.), ale mozesz je dostosowac do swoich potrzeb:

1. Przejdz do **Statusy zamowien** w sekcji "Sprzedaz - ustawienia".
2. Dodaj nowy status — podaj nazwe (np. "Czeka na czesci") i wybierz kolor.
3. Zdefiniuj dozwolone przejscia — np. z "Potwierdzone" mozna przejsc do "Czeka na czesci".
4. Kliknij **Zapisz**.

Nowy status bedzie od razu widoczny na liscie zamowien, w filtrach i na tablicy Kanban.

---

## Q: Co to sa pola niestandardowe?

Pola niestandardowe to dodatkowe informacje, ktore mozesz przyczepic do kazdego zamowienia. Jesli standardowe pola (klient, produkty, adres) nie wystarczaja, dodaj wlasne.

Dostepne typy pol:
- **Tekst** — dowolny tekst (np. "Numer paragonu", "Uwagi do pakowania")
- **Liczba** — wartosc liczbowa (np. "Koszt dodatkowy")
- **Lista wyboru** — wybor z predefiniowanych opcji (np. "Rodzaj opakowania": kartki, folia, pudelko)
- **Data** — pole daty (np. "Oczekiwana data dostawy")
- **Tak/Nie** — checkbox (np. "Prezent", "Faktura VAT")

Konfiguracja: **Pola niestandardowe** w sekcji "Sprzedaz - ustawienia".

Pola niestandardowe sa eksportowane do CSV i widoczne w szczegolach zamowienia.

---

## Q: Jak wystawic fakture?

OpenOMS integruje sie z trzema popularnymi polskimi serwisami fakturowania:

- **Fakturownia** (fakturownia.pl)
- **wFirma** (wfirma.pl)
- **InFakt** (infakt.pl)

Konfiguracja:
1. Przejdz do **Fakturowanie** w sekcji "Sprzedaz - ustawienia".
2. Wybierz serwis i podaj klucz API (znajdziesz go w ustawieniach konta w danym serwisie).
3. Zapisz.

Wystawianie:
1. Otworz zamowienie, kliknij **Wystaw fakture** — lub przejdz do **Faktury** i kliknij **Nowa faktura**.
2. Dane klienta i produkty sa wypelniane automatycznie z zamowienia.
3. Sprawdz dane, zatwierdz — faktura zostanie wyslana do serwisu fakturowania.

Mozesz tez skonfigurowac automatyczne wystawianie faktur przez reguly automatyzacji.

---

## Q: Jak generowac etykiety InPost?

1. Otworz zamowienie i kliknij **Utworz przesylke**.
2. Jako przewoznika wybierz **InPost**.
3. Uzupelnij dane: waga paczki, docelowy Paczkomat (mapa Paczkomatow jest wbudowana w OpenOMS).
4. Zapisz przesylke.
5. Na stronie przesylki kliknij **Generuj etykiete** — etykieta PDF zostanie pobrana.
6. Wydrukuj i przyklej na paczke.

Mozesz tez generowac etykiety zbiorczo: zaznacz kilka przesylek na liscie i kliknij **Drukuj etykiety**.

Wiecej: [Poradnik uzytkownika — Przesylki](poradnik-uzytkownika.md#7-przesylki)

---

## Q: Jak dziala synchronizacja z Allegro?

Po polaczeniu konta Allegro (OAuth2) system automatycznie:

- **Co kilka minut** pobiera nowe zamowienia z Allegro i dodaje je do listy zamowien
- **Synchronizuje stany magazynowe** — jesli zmienisz stan produktu w OpenOMS, zostanie zaktualizowany na Allegro
- **Importuje wiadomosci** od kupujacych
- **Aktualizuje statusy** — np. gdy nadasz przesylke, Allegro zostanie powiadomione

Status synchronizacji mozesz sprawdzic w **Synchronizacja** (sekcja "Monitoring").

Jesli chcesz wymusic natychmiastowa synchronizacje, przejdz do strony integracji Allegro i kliknij **Synchronizuj teraz**.

---

## Q: Czy moge eksportowac zamowienia do CSV/Excel?

Tak. Na stronie **Zamowienia** kliknij przycisk eksportu (ikona pobierania) w gornym pasku. Plik CSV zawiera:

- Numer zamowienia
- Dane klienta (imie, nazwisko, adres, email)
- Produkty, ilosci, ceny
- Status zamowienia i platnosci
- Tagi
- Pola niestandardowe
- Daty (utworzenia, ostatniej aktualizacji)

Eksport uwzglednia aktywne filtry — jesli filtrujesz po statusie "Wyslane", wyeksportuja sie tylko wyslane zamowienia.

Plik CSV mozna otworzyc w Excelu, Google Sheets lub dowolnym arkuszu kalkulacyjnym.

---

## Q: Jak dodac webhook?

1. Przejdz do **Webhooki** w sekcji "Powiadomienia" (wymaga roli Administratora).
2. Kliknij **Dodaj endpoint**.
3. Wypelnij:
   - **Nazwa** — np. "Slack powiadomienia"
   - **URL** — adres, na ktory maja byc wysylane dane (np. URL z Zapier, Make, czy Twojej aplikacji)
   - **Sekret** — klucz do podpisywania wiadomosci (HMAC-SHA256)
   - **Zdarzenia** — zaznacz, ktore zdarzenia Cie interesuja (np. "Zamowienie utworzone", "Status zamowienia zmieniony")
4. Kliknij **Zapisz**.

Kazde wywolanie webhooka jest logowane — przejdz do **Dostawy webhookow** w sekcji "Monitoring", aby zobaczyc historie wyslanych powiadomien, statusy HTTP i ewentualne bledy.

---

## Q: Jak zmienic haslo?

1. Przejdz do **Bezpieczenstwo** w sekcji "Ogolne" w menu bocznym.
2. W sekcji "Zmiana hasla" wpisz obecne haslo i nowe haslo.
3. Kliknij **Zmien haslo**.

Na tej samej stronie mozesz tez wlaczyc weryfikacje dwuetapowa (2FA/TOTP) — zeskanuj kod QR aplikacja uwierzytelniajaca (Google Authenticator, Authy).

---

## Q: Czy moje dane sa bezpieczne?

Tak. OpenOMS stosuje kilka warstw zabezpieczen:

- **Izolacja danych (multi-tenant RLS)** — kazda firma widzi tylko swoje dane. Nawet jesli wiele firm korzysta z tej samej instancji OpenOMS, dane sa scisle oddzielone na poziomie bazy danych (Row-Level Security w PostgreSQL).
- **Szyfrowanie danych wrażliwych** — klucze API integracji (Allegro, InPost itd.) sa szyfrowane algorytmem AES-256-GCM. Nigdy nie sa zwracane przez API — widoczna jest tylko informacja "klucz skonfigurowany".
- **JWT z Ed25519** — tokeny sesji podpisane kluczem kryptograficznym Ed25519.
- **Bezpieczne ciasteczka** — refresh token przechowywany w httpOnly cookie (niedostepny z JavaScript).
- **Weryfikacja dwuetapowa (2FA)** — opcjonalnie dla kazdego uzytkownika.
- **Webhooki podpisane HMAC-SHA256** — mozesz zweryfikowac autentycznosc przychodzacych danych.

---

## Q: Jak zglosic zwrot (RMA)?

1. Przejdz do **Zwroty** w menu bocznym.
2. Kliknij **Nowy zwrot**.
3. Wskazz zamowienie, ktorego dotyczy zwrot.
4. Podaj powod zwrotu i wybierz produkty do zwrotu.
5. Zapisz — zwrot otrzyma status "Zgloszone".

Nastepnie obsluz zwrot zmieniajac kolejne statusy: Zatwierdzone → Odebrane → Zwrocone (pieniadze oddane klientowi).

Klienci moga tez zglaszac zwroty samodzielnie przez formularz self-service — nie musza pisac maili.

Wiecej: [Poradnik uzytkownika — Zwroty](poradnik-uzytkownika.md#9-zwroty-rma)

---

## Q: Czy jest tryb ciemny?

Tak. Kliknij ikone ksiezyca/slonca w gornym pasku nawigacji, aby przelaczac miedzy trybem jasnym a ciemnym. Ustawienie jest zapamietywane w przegladarce.

---

## Q: Jak skontaktowac sie z supportem?

OpenOMS to projekt open-source. Mozesz uzyskac pomoc na kilka sposobow:

- **Discord** — dolacz do naszego serwera Discord (link w stopce strony), aby porozmawiac z innymi uzytkownikami i tworcami
- **GitHub Issues** — zglos blad lub zaproponuj nowa funkcje na GitHubie projektu
- **Dokumentacja** — przeczytaj [Poradnik uzytkownika](poradnik-uzytkownika.md)

Jesli korzystasz z komercyjnej wersji hostowanej, skontaktuj sie z dostawca uslugi.

---

## Q: Czy moge podlaczyc Amazon / WooCommerce / eBay?

Tak. OpenOMS obsluguje wiele kanalow sprzedazy:

- **Allegro** — pelna integracja (zamowienia, oferty, wiadomosci, zwroty, finanse)
- **Amazon** — integracja przez SP-API (zamowienia, produkty)
- **eBay** — import zamowien i synchronizacja ofert
- **Kaufland** — import zamowien
- **WooCommerce** — synchronizacja zamowien i produktow
- **Empik** / **Erli** / **OLX** — import zamowien
- **Shoper** / **PrestaShop** / **Shopify** — integracja sklepu internetowego

Kazda integracje konfigurujesz w sekcji **Kanaly sprzedazy** w menu bocznym. Przejdz do wybranego marketplace'u i postepuj zgodnie z instrukcjami na ekranie.

Zamowienia ze wszystkich podlaczonych kanalow trafiaja na wspolna liste zamowien w OpenOMS.

---

## Q: Jak dziala automatyzacja zamowien?

Automatyzacja pozwala definiowac reguly "jezeli-to" dla zamowien i innych zdarzen w systemie.

**Przykladowe reguly:**

- *Gdy nowe zamowienie z Allegro za > 200 PLN → dodaj tag "VIP" i wyslij maila do magazynu*
- *Gdy status zmieni sie na "Wyslane" → wyslij SMS do klienta z numerem sledzenia*
- *Gdy zamowienie nie zostanie oplacone w 48h → zmien status na "Anulowane"*
- *Gdy produkt osiagnie stan 0 → wyslij alert do administratora*

Konfiguracja:
1. Przejdz do **Automatyzacja** w sekcji "Narzedzia".
2. Kliknij **Nowa regula**.
3. Wybierz wyzwalacz (np. "Zamowienie utworzone"), dodaj warunki i ustaw akcje.
4. Zapisz — regula zacznie dzialac natychmiast.

Wiecej zaawansowanych automatyzacji mozesz tworzyc w **Workflow Builder** (sekcja "Narzedzia") — wizualny edytor procesow.

---

*Nie znalazles odpowiedzi? Sprawdz [Poradnik uzytkownika](poradnik-uzytkownika.md) lub napisz do nas na [Discordzie](https://discord.gg/openoms).*
