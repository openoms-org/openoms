# Deploy OpenOMS Landing Page na Cloudflare Pages

## Co masz gotowe

Folder `deploy/` zawiera wszystko, co potrzebujesz:

- `index.html` — landing page z formularzem waitlisty (podpięty do Formspree)
- 6 obrazów PNG (fallback) + 6 obrazów WebP (lekkie, ładowane priorytetowo)

Waitlista działa — formularz wysyła dane na Twój endpoint Formspree (`xkovgrgb`).

---

## Krok po kroku

### 1. Zaloguj się do Cloudflare Dashboard

Wejdź na [dash.cloudflare.com](https://dash.cloudflare.com) i zaloguj się na konto, na którym kupiłeś openoms.org.

### 2. Utwórz projekt Cloudflare Pages

1. W lewym menu kliknij **Workers & Pages**
2. Kliknij **Create** (niebieski przycisk)
3. Wybierz zakładkę **Pages**
4. Kliknij **Upload assets** (nie potrzebujesz łączyć z GitHubem — wystarczy wrzucić pliki)
5. Nazwij projekt: `openoms` (albo `openoms-org`)
6. Przeciągnij **cały folder `deploy/`** do uploadu (albo kliknij "select" i wybierz folder)
7. Poczekaj aż wszystkie pliki się załadują
8. Kliknij **Deploy site**

Po deployu dostaniesz tymczasowy URL typu `openoms.pages.dev` — możesz od razu sprawdzić, czy strona działa.

### 3. Podepnij domenę openoms.org

1. W dashboardzie projektu Pages, przejdź do **Custom domains**
2. Kliknij **Set up a custom domain**
3. Wpisz: `openoms.org`
4. Cloudflare automatycznie doda rekord DNS (bo domena jest na tym samym koncie)
5. Kliknij **Activate domain**
6. Powtórz dla `www.openoms.org` (opcjonalnie, ale warto)

DNS propagacja na Cloudflare jest natychmiastowa (bo domena jest już u nich). Strona powinna działać pod openoms.org w ciągu kilku minut.

### 4. Sprawdź czy działa

- Wejdź na `https://openoms.org`
- Sprawdź czy strona się ładuje poprawnie
- Wpisz testowy email w formularz waitlisty
- Sprawdź w Formspree ([formspree.io/f/xkovgrgb](https://formspree.io/f/xkovgrgb)) czy email dotarł
- Usuń testowy wpis z Formspree

---

## Przyszłe aktualizacje

Gdy będziesz chciał zmienić coś na stronie:

1. Edytuj `index.html` (lub obrazy) w folderze `deploy/`
2. W dashboardzie Cloudflare Pages → Twój projekt → **Create new deployment**
3. Wrzuć ponownie cały folder `deploy/`
4. Nowa wersja jest live w kilka sekund

---

## Przydatne linki

- Cloudflare Dashboard: [dash.cloudflare.com](https://dash.cloudflare.com)
- Formspree (zgłoszenia z waitlisty): [formspree.io](https://formspree.io)
- Cloudflare Pages docs: [developers.cloudflare.com/pages](https://developers.cloudflare.com/pages)
