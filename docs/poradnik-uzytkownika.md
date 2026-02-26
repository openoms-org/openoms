# OpenOMS — Poradnik uzytkownika

## Spis tresci

1. [Wstep](#1-wstep)
2. [Rejestracja i logowanie](#2-rejestracja-i-logowanie)
3. [Konfiguracja firmy](#3-konfiguracja-firmy)
4. [Polaczenie z Allegro](#4-polaczenie-z-allegro)
5. [Zamowienia](#5-zamowienia)
6. [Produkty](#6-produkty)
7. [Przesylki](#7-przesylki)
8. [Faktury](#8-faktury)
9. [Zwroty (RMA)](#9-zwroty-rma)
10. [Ustawienia](#10-ustawienia)
11. [Skroty klawiszowe](#11-skroty-klawiszowe)

---

## 1. Wstep

OpenOMS to nowoczesny system zarzadzania zamowieniami (Order Management System) stworzony z mysla o polskich sprzedawcach e-commerce. Jesli sprzedajesz na Allegro, Amazonie, eBayu, przez WooCommerce lub inne kanaly — OpenOMS pomoze Ci ogarnac zamowienia, przesylki, produkty i faktury w jednym miejscu.

**Co daje Ci OpenOMS?**

- Centralny widok wszystkich zamowien z roznych kanalow sprzedazy
- Automatyczna synchronizacja z Allegro (zamowienia, oferty, wiadomosci)
- Generowanie etykiet kurierskich (InPost, DHL, DPD, GLS, UPS, Poczta Polska, FedEx)
- Integracja z systemami fakturowania (Fakturownia, wFirma, InFakt)
- Obsluga zwrotow (RMA) z pelnym cyklem zycia
- Reguly automatyzacji — np. automatyczna zmiana statusu, wysylka maila do klienta
- Wiele magazynow, inwentaryzacja, kontrola stanow
- Raporty sprzedazy, trendow i prognoz

OpenOMS jest oprogramowaniem open-source na licencji Elastic License 2.0 (ELv2) — mozesz korzystac z niego za darmo, hostujac na wlasnym serwerze.

**Glowne zalety w porownaniu z innymi systemami:**

- Brak miesiecznych oplat ani limitow zamowien
- Pelna kontrola nad danymi — hostujesz u siebie
- Interfejs calkowicie po polsku
- Integracja z polskimi systemami: Allegro, InPost, Fakturownia, KSeF
- Tryb ciemny, PWA (aplikacja mobilna), skroty klawiszowe

---

## 2. Rejestracja i logowanie

### Tworzenie konta

Aby zaczac korzystac z OpenOMS, musisz miec konto uzytkownika przypisane do organizacji (tzw. "tenant" — Twoja firma w systemie). Jesli jestes pierwszym uzytkownikiem, administrator systemu tworzy organizacje i pierwsze konto.

Kazda organizacja ma unikalny **slug** — krotka nazwe identyfikujaca firme (np. `mojsklep`, `allegro-hurtownia`). Slug bedzie potrzebny przy logowaniu.

### Logowanie

1. Wejdz na strone logowania OpenOMS.
2. Wypelnij trzy pola:
   - **Slug organizacji** — krotka nazwa Twojej firmy (np. `mojsklep`)
   - **Email** — Twoj adres email
   - **Haslo** — Twoje haslo
3. Kliknij **Zaloguj sie**.

[Screenshot: Ekran logowania z trzema polami — slug, email, haslo]

**Uwaga:** Ten sam adres email moze istniec w roznych organizacjach. Slug sluzy do rozroznienia, do ktorej firmy chcesz sie zalogowac.

### Weryfikacja dwuetapowa (2FA)

Jesli masz wlaczona weryfikacje dwuetapowa (TOTP), po wpisaniu hasla zobaczysz dodatkowe pole na 6-cyfrowy kod z aplikacji uwierzytelniajac (np. Google Authenticator, Authy). Kod zostanie automatycznie wyslany po wpisaniu 6 cyfr.

Wlaczenie 2FA: Przejdz do **Bezpieczenstwo** w menu bocznym.

[Screenshot: Ekran weryfikacji 2FA]

---

## 3. Konfiguracja firmy

Po pierwszym zalogowaniu warto uzupelnic dane firmowe. Beda one wykorzystywane na fakturach, etykietach i w korespondencji z klientami.

### Dane firmy

1. Przejdz do **Firma** w sekcji "Ogolne" w menu bocznym.
2. Uzupelnij:
   - **Nazwa firmy**
   - **NIP**
   - **Adres** (ulica, miasto, kod pocztowy)
   - **Telefon kontaktowy**
   - **Email firmowy**
3. Mozesz tez wgrac **logo firmy** — bedzie widoczne na fakturach i w interfejsie.
4. Kliknij **Zapisz**.

[Screenshot: Formularz danych firmy z wypelnionymi polami]

### Dodawanie uzytkownikow

Jesli masz zespol, mozesz dodac kolejnych pracownikow:

1. Przejdz do **Uzytkownicy** w sekcji "Ogolne".
2. Kliknij **Dodaj uzytkownika**.
3. Podaj email, imie, nazwisko i wybierz role:
   - **Wlasciciel** — pelny dostep do wszystkiego
   - **Administrator** — zarzadza ustawieniami i uzytkownikami
   - **Czlonek** — podstawowy dostep do zamowien i produktow
4. Nowy uzytkownik otrzyma email z danymi dostepu.

Mozesz tez tworzyc wlasne role z indywidualnie dobranymi uprawnieniami — przejdz do **Role** w sekcji "Ogolne".

---

## 4. Polaczenie z Allegro

Allegro to najpopularniejszy marketplace w Polsce, dlatego integracja z nim jest kluczowa. OpenOMS laczy sie z Allegro przez oficjalne API (OAuth2).

### Krok po kroku

1. Przejdz do **Allegro** w sekcji "Kanaly sprzedazy" w menu bocznym.
2. Kliknij **Polacz z Allegro**.
3. Zostaniesz przekierowany na strone Allegro — zaloguj sie i zaakceptuj uprawnienia dla OpenOMS.
4. Po zatwierdzeniu wrocisz do OpenOMS. Status integracji zmieni sie na **Aktywna**.

[Screenshot: Strona integracji Allegro ze statusem "Aktywna"]

### Co synchronizuje sie automatycznie?

Po polaczeniu OpenOMS automatycznie:

- **Importuje zamowienia** — nowe zamowienia z Allegro pojawia sie w zakladce Zamowienia
- **Synchronizuje oferty** — Twoje oferty Allegro widoczne w zakladce Oferty
- **Pobiera wiadomosci** — korespondencja z kupujacymi dostepna w zakladce Wiadomosci
- **Aktualizuje stany magazynowe** — zmiany w OpenOMS moga byc przesylane do Allegro
- **Obsluguje zwroty** — zwroty Allegro widoczne w dedykowanej zakladce

### Podstrony Allegro

Po polaczeniu masz dostep do dodatkowych zakladek:

- **Oferty** — zarzadzanie ofertami Allegro
- **Katalog** — przegladanie katalogu Allegro
- **Promocje** — zarzadzanie promocjami
- **Wiadomosci** — korespondencja z kupujacymi
- **Zwroty** — zwroty Allegro
- **Spory** — obsluga sporow
- **Dostawa** — ustawienia dostaw
- **Polityki** — polityki sprzedazy
- **Finanse** — rozliczenia
- **Oceny** — oceny kupujacych
- **Przesylki** — przesylki "Wysylam z Allegro"

[Screenshot: Menu boczne z rozwinietymi podstronami Allegro]

### Inne marketplace'y

OpenOMS obsluguje takze: Amazon, eBay, Kaufland, Empik, Erli, OLX, WooCommerce, Shoper, PrestaShop i Shopify. Kazdy z nich konfigurujesz w sekcji **Kanaly sprzedazy**.

---

## 5. Zamowienia

Zamowienia to serce OpenOMS. Tutaj widzisz wszystkie zamowienia — niezaleznie od tego, czy przyszly z Allegro, Amazona, czy zostaly dodane recznie.

### Lista zamowien

Przejdz do **Zamowienia** w menu bocznym. Zobaczysz tabele ze wszystkimi zamowieniami.

[Screenshot: Lista zamowien z filtrami i akcjami]

**Widoki:**

- **Tabela** — klasyczny widok tabelaryczny z sortowaniem kolumn
- **Kanban** — widok tablicowy pogrupowany po statusach (przeciagaj zamowienia miedzy kolumnami)

Przelaczaj sie miedzy widokami ikonami w prawym gornym rogu listy.

### Filtry i wyszukiwanie

Nad tabela znajdziesz filtry:

- **Status** — np. Nowe, W realizacji, Wysiane, Dostarczone
- **Zrodlo** — Allegro, Amazon, Reczne, WooCommerce itd.
- **Status platnosci** — Oczekuje, Oplacone, Czesciowo, Zwrocone
- **Priorytet** — Pilne, Wysoki, Normalny, Niski
- **Tag** — filtrowanie po tagach
- **Wyszukiwanie** — szukaj po numerze zamowienia, nazwisku klienta, adresie email

### Statusy zamowien

Domyslne statusy:

| Status | Opis |
|--------|------|
| Nowe | Zamowienie wlasnie wplynelo |
| Potwierdzone | Zamowienie zostalo potwierdzone |
| W realizacji | Zamowienie jest kompletowane |
| Gotowe do wysylki | Paczka czeka na kuriera |
| Wyslane | Paczka zostala nadana |
| W transporcie | Paczka jest w drodze |
| W doreczeniu | Kurier doreczna paczke |
| Dostarczone | Klient odebral paczke |
| Zakonczone | Transakcja zamknieta |
| Wstrzymane | Zamowienie tymczasowo wstrzymane |
| Anulowane | Zamowienie anulowane |
| Zwrocone | Pieniadze zwrocone klientowi |

Mozesz tworzyc wlasne statusy — patrz rozdzial [Ustawienia](#10-ustawienia).

### Zmiana statusu

1. Kliknij zamowienie, aby otworzyc szczegoly.
2. Na gorze zobaczysz aktualny status i dostepne przejscia.
3. Kliknij nowy status — zmiana zostanie zapisana natychmiast.

**Zmiana zbiorcza:** Zaznacz kilka zamowien na liscie (checkboxy) i uzyj przycisku **Akcje zbiorcze** nad tabela, aby zmienic status wielu zamowien na raz.

### Szczegoly zamowienia

Po kliknieciu zamowienia zobaczysz:

- Dane klienta (imie, nazwisko, adres, email, telefon)
- Lista produktow w zamowieniu (nazwy, ilosci, ceny)
- Informacje o platnosci (metoda, status)
- Historia statusow (os czasu audytu)
- Przesylki powiazane z zamowieniem
- Pola niestandardowe (jesli zdefiniowane)
- Tagi
- Notatki wewnetrzne

### Scalanie i rozdzielanie zamowien

- **Scalanie** — jesli ten sam klient zlozyl kilka zamowien, mozesz je polaczyc w jedno. Zaznacz zamowienia i kliknij **Scal**.
- **Rozdzielanie** — jesli zamowienie ma produkty z roznych magazynow, mozesz je rozdzielic na osobne zamowienia.

### Eksport CSV

Kliknij przycisk **Eksportuj** (ikona pobierania) nad tabela zamowien. Plik CSV zawiera wszystkie widoczne zamowienia z uwzglednieniem aktualnych filtrow.

### Tworzenie recznego zamowienia

Kliknij **Nowe zamowienie** lub uzyj skrotu **Ctrl+N**. Wypelnij dane klienta, dodaj produkty i zapisz.

### Import zamowien z CSV

Oprócz automatycznego importu z marketplace'ow mozesz tez importowac zamowienia z pliku CSV:

1. Przejdz do **Import** w sekcji "Sprzedaz".
2. Wgraj plik CSV z danymi zamowien.
3. Zmapuj kolumny pliku na pola w OpenOMS.
4. Zatwierdz import.

Jest to przydatne jesli przenosisz sie z innego systemu lub masz zamowienia z kanalow nieobslugiwanych przez integracje.

### Metody platnosci

OpenOMS obsluguje nastepujace metody platnosci: przelew bankowy, pobranie (COD), karta platnicza, PayU, Przelewy24 oraz BLIK. Metoda platnosci jest przypisywana do kazdego zamowienia i widoczna w szczegolach.

---

## 6. Produkty

### Lista produktow

Przejdz do **Produkty** w menu bocznym. Zobaczysz liste wszystkich produktow z mozliwoscia filtrowania, sortowania i wyszukiwania.

[Screenshot: Lista produktow z filtrami kategorii i tagow]

**Filtry:**

- **Wyszukiwanie** — szukaj po nazwie lub SKU
- **Kategoria** — filtruj po kategorii produktu
- **Tag** — filtruj po tagach

### Dodawanie produktu

1. Kliknij **Nowy produkt** lub uzyj skrotu **Ctrl+Shift+N**.
2. Wypelnij formularz:
   - **Nazwa** — nazwa produktu
   - **SKU** — unikalny kod produktu
   - **Cena** — cena sprzedazy
   - **Ilosc na stanie** — aktualny stan magazynowy
   - **Opis krotki** — zwiezly opis widoczny na listach
   - **Opis dlugi** — pelny opis produktu
   - **Waga** — waga w kilogramach
   - **Wymiary** — szerokosc, wysokosc, dlugosc, w centymetrach
   - **Zdjecia** — wgraj zdjecia produktu (URL lub plik z dysku)
   - **Kategoria** — przypisz kategorie
   - **Tagi** — dodaj tagi do grupowania produktow
3. Kliknij **Zapisz**.

[Screenshot: Formularz dodawania produktu]

### Warianty produktu

Jesli produkt wystepuje w roznych wariantach (np. rozmiar, kolor):

1. Otworz produkt i przejdz do zakladki **Warianty**.
2. Dodaj warianty z indywidualnymi cenami, SKU i stanami magazynowymi.

### Kategorie produktow

Zdefiniuj wlasne kategorie w **Kategorie** (sekcja "Katalog" w menu):

1. Dodaj kategorie z nazwa i kolorem.
2. Przypisuj kategorie do produktow w formularzu edycji.
3. Filtruj produkty po kategorii na liscie.

### Import produktow z CSV

Jesli masz duzo produktow, mozesz je zaimportowac hurtowo:

1. Przejdz do **Import produktow** w sekcji "Katalog".
2. Pobierz szablon CSV.
3. Uzupelnij plik danymi produktow.
4. Wgraj plik — system zaimportuje produkty i pokaze raport z ewentualnymi bledami.

[Screenshot: Strona importu CSV z przeciagnietym plikiem]

### Kategoryzacja AI

Jesli masz wiele produktow bez kategorii, zaznacz je na liscie i kliknij **Kategoryzuj AI** — system automatycznie zaproponuje kategorie na podstawie nazw i opisow.

### Tagi produktow

Tagi to elastyczny sposob grupowania produktow. W przeciwienstwie do kategorii (jeden produkt = jedna kategoria), produkt moze miec wiele tagow.

Przyklady tagow: `bestseller`, `promocja`, `sezon-letni`, `fragile`.

Tagi dodajesz w formularzu edycji produktu lub bezposrednio w tabeli (edycja inline). Mozesz tez filtrowac liste produktow po tagach.

### Oferty na marketplace'ach

Jesli masz polaczone konto Allegro, mozesz z poziomu produktu zobaczyc powiazane oferty. Przejdz do zakladki **Oferty** w szczegolach produktu, aby zobaczyc status oferty na Allegro i szybko do niej przejsc.

### Usuwanie tla ze zdjec

OpenOMS oferuje narzedzie AI do usuwania tla ze zdjec produktow. Przejdz do **Usuwanie tla** w sekcji "Narzedzia", wgraj zdjecie i pobierz wersje z bialym tlem — idealne do ofert na Allegro.

---

## 7. Przesylki

### Tworzenie przesylki

Przesylke najczesciej tworzysz z poziomu zamowienia:

1. Otworz zamowienie.
2. Kliknij **Utwórz przesylke**.
3. Wybierz przewoznika:
   - **InPost** — Paczkomaty i kurier
   - **DHL** — kurier DHL
   - **DPD** — kurier DPD
   - **GLS** — kurier GLS
   - **UPS** — kurier UPS
   - **Poczta Polska** — przesylki pocztowe
   - **Orlen Paczka** — paczki Orlen
   - **FedEx** — kurier FedEx
4. Uzupelnij dane przesylki (waga, wymiary, punkt odbioru dla Paczkomatow).
5. Kliknij **Zapisz**.

[Screenshot: Dialog tworzenia przesylki z wyborem przewoznika]

Mozesz tez utworzyc przesylke z poziomu listy przesylek (**Przesylki** w menu bocznym) klikajac **Nowa przesylka**.

### Lista przesylek

W zakladce **Przesylki** widzisz wszystkie przesylki z filtrami:

- **Status** — Utworzona, Etykieta gotowa, Odebrana, W transporcie, Dostarczona itd.
- **Przewoznik** — InPost, DHL, DPD, GLS, UPS itd.

### Generowanie etykiet

1. Otworz przesylke.
2. Kliknij **Generuj etykiete**.
3. Etykieta zostanie pobrana jako PDF — wydrukuj ja i przyklej na paczke.

**Etykiety zbiorcze:** Zaznacz kilka przesylek na liscie i kliknij **Drukuj etykiety** — wszystkie etykiety zostaną pobrane w jednym pliku PDF.

### Sledzenie przesylki

Numer sledzenia (tracking number) pojawia sie automatycznie po wygenerowaniu etykiety. OpenOMS automatycznie odpytuje API przewoznikow o aktualizacje statusu.

### Statusy przesylek

| Status | Opis |
|--------|------|
| Utworzona | Przesylka zarejestrowana w systemie |
| Etykieta gotowa | Etykieta wygenerowana, czeka na nadanie |
| Odebrana | Kurier odebral paczke |
| W transporcie | Paczka w drodze |
| W doreczeniu | Kurier doreczna paczke |
| Dostarczona | Paczka dostarczona |
| Zwrocona | Paczka wrocila do nadawcy |
| Nieudana | Wystapil problem z doreczeniem |

### Stacja pakowania

Przejdz do **Pakowanie** w menu — znajdziesz tam widok do skanowania zamowien kodem kreskowym i szybkiego pakowania.

### Pick & Pack

Dla wiekszych magazynow OpenOMS oferuje modul **Pick & Pack** (sekcja "Magazyn"):

1. System generuje liste kompletacji (picking list) na podstawie zamowien gotowych do realizacji.
2. Magazynier skanuje produkty i potwierdza skompletowanie.
3. Na etapie pakowania skanuje paczke i zamowienie — system weryfikuje zgodnosc.

Dzieki temu minimalizujesz bledy pakowania i przyspieszasz obsluge zamowien.

---

## 8. Faktury

OpenOMS integruje sie z popularnymi polskimi systemami fakturowania.

### Konfiguracja

1. Przejdz do **Fakturowanie** w sekcji "Sprzedaz - ustawienia".
2. Wybierz dostawce: **Fakturownia**, **wFirma** lub **InFakt**.
3. Podaj klucz API z wybranego systemu.
4. Zapisz ustawienia.

[Screenshot: Ustawienia integracji fakturowania]

### Wystawianie faktur

1. Przejdz do **Faktury** w menu bocznym.
2. Kliknij **Nowa faktura** lub wejdz w zamowienie i kliknij **Wystaw fakture**.
3. Dane klienta i produkty beda uzupelnione automatycznie na podstawie zamowienia.
4. Sprawdz dane i zatwierdz.

Faktura zostanie wyslana do systemu fakturowania i bedzie widoczna na liscie faktur.

### KSeF (Krajowy System e-Faktur)

Jesli chcesz korzystac z KSeF, przejdz do **KSeF** w sekcji "Sprzedaz - ustawienia" i skonfiguruj polaczenie. OpenOMS umozliwia automatyczne przesylanie faktur do KSeF.

---

## 9. Zwroty (RMA)

### Zgloszenie zwrotu

1. Przejdz do **Zwroty** w menu bocznym.
2. Kliknij **Nowy zwrot**.
3. Wybierz zamowienie, ktorego dotyczy zwrot.
4. Podaj powod zwrotu i liste produktow do zwrotu.
5. Zapisz.

[Screenshot: Formularz nowego zwrotu]

Klienci moga tez zglaszac zwroty samodzielnie przez dedykowany formularz (self-service returns).

### Statusy zwrotow

| Status | Opis |
|--------|------|
| Zgloszone | Zwrot zostal zgloszony |
| Zatwierdzone | Zwrot zaakceptowany, czekamy na paczke |
| Odebrane | Produkt wrocil do magazynu |
| Zwrocone | Pieniadze zwrocone klientowi |
| Odrzucone | Zwrot odrzucony |
| Anulowane | Zwrot anulowany |

### Obsluga zwrotu

1. Na liscie zwrotow kliknij zglosznie.
2. Zmien status w zaleznosci od etapu:
   - **Zatwierdz** — jesli akceptujesz zwrot
   - **Odebrane** — gdy produkt wroci do Ciebie
   - **Zwroc pieniadze** — po dokonaniu zwrotu srodkow
   - **Odrzuc** — jesli zwrot jest niezasadny
3. Kazda zmiana statusu jest zapisywana w dzienniku aktywnosci.

---

## 10. Ustawienia

### Statusy zamowien

Przejdz do **Statusy zamowien** w sekcji "Sprzedaz - ustawienia".

Mozesz:

- Dodawac nowe statusy z wlasna nazwa i kolorem
- Usuwac zbedne statusy
- Definiowac dozwolone przejscia miedzy statusami (np. "Nowe" moze przejsc do "Potwierdzone" lub "Anulowane", ale nie od razu do "Dostarczone")

[Screenshot: Konfiguracja statusow zamowien z kolorowymi etykietami]

### Pola niestandardowe

Przejdz do **Pola niestandardowe** w sekcji "Sprzedaz - ustawienia".

Pola niestandardowe pozwalaja dodawac wlasne informacje do zamowien. Dostepne typy:

- **Tekst** — dowolny tekst
- **Liczba** — wartosc liczbowa
- **Lista wyboru** — wybor z predefiniowanych opcji
- **Data** — pole daty
- **Tak/Nie** — pole logiczne (checkbox)

Przyklad: Mozesz dodac pole "Numer paragonu" typu Tekst albo pole "Prezent" typu Tak/Nie.

Pola niestandardowe sa widoczne w szczegolach zamowienia i eksportowane do CSV.

### Webhooki

Przejdz do **Webhooki** w sekcji "Powiadomienia".

Webhooki pozwalaja powiadamiac zewnetrzne systemy o zdarzeniach w OpenOMS. Mozesz np. wyslac powiadomienie do Slacka, gdy pojawi sie nowe zamowienie.

Dostepne zdarzenia:

- Zamowienie utworzone / status zmieniony / usuniete
- Produkt utworzony / zaktualizowany / usuniety
- Przesylka utworzona / zaktualizowana

Kazdy webhook jest podpisany kluczem HMAC-SHA256 — Twoj system moze zweryfikowac, ze dane pochodza z OpenOMS.

Przejdz do **Dostawy webhookow** w sekcji "Monitoring", aby zobaczyc historie wyslanych webhookow i ewentualne bledy.

### Powiadomienia email i SMS

Przejdz do **Powiadomienia** w sekcji "Powiadomienia".

Mozesz skonfigurowac:

- **Powiadomienia email** — automatyczne maile do klientow (np. potwierdzenie zamowienia, zmiana statusu, numer sledzenia)
- **Powiadomienia SMS** — wiadomosci SMS do klientow

Szablony sa w jezyku polskim i mozna je personalizowac.

### Automatyzacja

Przejdz do **Automatyzacja** w sekcji "Narzedzia".

Reguly automatyzacji wykonuja akcje gdy spelnione sa warunki:

- **Wyzwalacz** — zdarzenie, np. "Zamowienie utworzone", "Status zmieniony na Wyslane"
- **Warunki** — np. "Zrodlo = Allegro", "Kwota > 100 PLN"
- **Akcje** — np. "Zmien status", "Wyslij email", "Wyslij SMS", "Dodaj tag"

Przyklad: Gdy zamowienie z Allegro zmieni status na "Wyslane", automatycznie wyslij klientowi SMS z numerem sledzenia.

[Screenshot: Formularz nowej reguly automatyzacji]

### Magazyny

Przejdz do **Magazyny** w sekcji "Magazyn".

Jesli masz wiecej niz jeden magazyn, mozesz je zdefiniowac osobno. Kazdy magazyn ma:

- Nazwe i adres
- Wlasne stany magazynowe
- Dokumenty magazynowe (PZ, WZ, MM)

### Inwentaryzacja

Przejdz do **Inwentaryzacja** w sekcji "Magazyn".

Mozesz przeprowadzac inwentaryzacje:

1. Utworz nowa inwentaryzacje.
2. Skanuj produkty lub wpisz ilosci recznie.
3. Porownaj stany systemowe z rzeczywistymi.
4. Zatwierdz roznice — stany zostana zaktualizowane.

### Kontrola magazynowa

Przejdz do **Kontrola magazynowa** w sekcji "Magazyn", aby ustawic polityki dotyczace stanow magazynowych — np. blokowanie sprzedazy przy zerowym stanie, alerty niskiego stanu.

### Szablony druku

Przejdz do **Szablony druku** w sekcji "Katalog".

Mozesz tworzyc wlasne szablony do drukowania:

- Etykiet produktowych z kodem kreskowym
- Listow przewozowych
- Dokumentow pakowania
- Faktur pro forma

Szablony sa edytowalne i moga zawierac logo Twojej firmy.

### Cenniki (B2B)

Przejdz do **Cenniki** w sekcji "Sprzedaz - ustawienia".

Jesli prowadzisz sprzedaz hurtowa, mozesz definiowac rozne cenniki dla roznych grup klientow. Kazdy cennik zawiera indywidualne ceny produktow — klient B2B widzi swoje ceny, a klient detaliczny inne.

### Klienci

Przejdz do **Klienci** w sekcji "Sprzedaz".

OpenOMS prowadzi baze klientow automatycznie — kazde zamowienie tworzy lub aktualizuje profil klienta. W profilu klienta zobaczysz:

- Dane kontaktowe (imie, nazwisko, email, telefon)
- Adresy dostawy
- Historie zamowien
- Laczna wartosc zamowien

Mozesz tez segmentowac klientow (**Segmenty** w menu) — np. "VIP klienci" z zamowieniami powyzej 1000 PLN, albo "Nieaktywni" bez zamowien od 6 miesiecy.

### Waluty

Przejdz do **Waluty** w sekcji "Narzedzia".

Jesli sprzedajesz miedzynarodowo, mozesz wlaczyc obsluge wielu walut (EUR, GBP, USD itd.). Kursy wymiany sa aktualizowane automatycznie.

### Dostawcy i zamowienia zakupu

Przejdz do **Dostawcy** i **Zamowienia zakupu** w sekcji "Narzedzia".

Jesli kupujesz towary od dostawcow, mozesz zarzadzac cala sciezka zakupowa:

1. Dodaj dostawcow z danymi kontaktowymi i cennikach.
2. Twórz zamowienia zakupu (purchase orders) i wysylaj je dostawcom.
3. Odnotowuj przyjecia towaru — stany magazynowe zaktualizuja sie automatycznie.

---

## 11. Skroty klawiszowe

OpenOMS obsluguje skroty klawiszowe, ktore przyspieszaja prace:

| Skrot | Akcja |
|-------|-------|
| **Ctrl+K** (lub Cmd+K na Mac) | Otwiera palete polecen — szybkie wyszukiwanie stron, zamowien, produktow |
| **Ctrl+N** | Nowe zamowienie |
| **Ctrl+Shift+N** | Nowy produkt |
| **Alt+1** | Przejdz do Pulpitu |
| **Alt+2** | Przejdz do Zamowien |
| **Alt+3** | Przejdz do Produktow |

[Screenshot: Paleta polecen (Ctrl+K) z wyszukiwaniem]

### Paleta polecen (Ctrl+K)

Paleta polecen to najszybszy sposob nawigacji w OpenOMS. Po otwarciu zacznij pisac:

- Nazwe strony (np. "Przesylki", "Ustawienia")
- Nazwe funkcji (np. "Import", "Automatyzacja")
- Numer zamowienia

System podpowie pasujace wyniki — kliknij lub nacisnij Enter, aby przejsc.

---

## Pulpit (Dashboard)

Po zalogowaniu trafiasz na Pulpit — glowny ekran z podsumowaniem Twojego biznesu:

- **Wykresy przychodow** — dzienny, tygodniowy i miesieczny przychod
- **Trendy sprzedazy** — porownanie z poprzednim okresem
- **Ostatnie zamowienia** — lista najnowszych zamowien z mozliwoscia szybkiego przejscia do szczegolów
- **Podsumowanie statusow** — ile zamowien czeka na realizacje, ile jest w drodze itd.

[Screenshot: Pulpit z wykresami przychodow i lista ostatnich zamowien]

Zaawansowane raporty (dostepne dla administratorow) znajdziesz w sekcji **Raporty**:

- **Raporty sprzedazy** — przychody, marze, top produkty
- **Prognoza popytu** — AI przewiduje zapotrzebowanie na produkty
- **Slad weglowy** — raport emisji CO2 z przesylek
- **Raport VAT OSS** — dla sprzedazy miedzynarodowej w UE

---

## Przydatne wskazowki

- **Tryb ciemny** — kliknij ikone ksiezyca/slonca w gornym pasku, aby przelaczac miedzy jasnym a ciemnym motywem.
- **Gestosc tabeli** — przy listach zamowien, produktow i przesylek znajdziesz przelacznik gestosci wyswietlania (kompaktowy / normalny).
- **Sortowanie kolumn** — klikaj naglowki kolumn w tabelach, aby sortowac dane rosnaco lub malejaco.
- **Edycja w tabeli** — niektorych pol mozesz edytowac bezposrednio w tabeli, bez otwierania szczegolowego widoku.
- **Dziennik aktywnosci** — przejdz do **Dziennik aktywnosci** w sekcji "Monitoring", aby zobaczyc kto i kiedy zmienial zamowienia, produkty i ustawienia (dostep tylko dla administratorow).
- **Raporty** — przejdz do **Raporty** w menu, aby zobaczyc wykresy przychodow, trendow sprzedazy i prognoz popytu.
- **PWA** — OpenOMS mozna zainstalowac jako aplikacje na telefonie (Progressive Web App). W przegladarce mobilnej wybierz "Dodaj do ekranu glownego".

---

*Ten poradnik bedzie aktualizowany wraz z rozwojem OpenOMS. Jesli masz pytania, sprawdz [FAQ dla sprzedawcow](faq-sprzedawcy.md) lub skontaktuj sie z nami na [Discordzie](https://discord.gg/openoms).*
