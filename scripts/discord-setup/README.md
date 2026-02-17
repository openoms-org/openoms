# OpenOMS Discord Server Setup

Jednorazowy skrypt konfigurujący serwer Discord z kanałami, rolami, uprawnieniami, politykami bezpieczeństwa i community features.

## Wymagania

- Node.js 18+
- Bot Discord z uprawnieniami Administratora
- Włączone intenty: Server Members, Message Content

## Jak użyć

### 1. Stwórz bota Discord

1. Wejdź na https://discord.com/developers/applications
2. Kliknij **New Application** → nazwa: `OpenOMS Bot`
3. Zakładka **Bot**:
   - Skopiuj **Token** (będzie potrzebny)
   - Włącz **Server Members Intent**
   - Włącz **Message Content Intent**
4. Zakładka **OAuth2** → **URL Generator**:
   - Scopes: `bot`
   - Bot Permissions: `Administrator`
   - Skopiuj wygenerowany URL i otwórz go — dodaj bota do swojego serwera

### 2. Uruchom skrypt

```bash
cd scripts/discord-setup
npm install

DISCORD_BOT_TOKEN=twoj_token DISCORD_GUILD_ID=id_serwera npm run setup
```

> **Guild ID**: Włącz Developer Mode w Discord (Settings → Advanced), kliknij prawym na serwer → Copy Server ID.

### 3. Skonfiguruj GitHub → Discord webhook

1. W Discord: kanał `#github-feed` → Ustawienia → Integracje → Webhooky → Nowy webhook
2. Skopiuj **Webhook URL**
3. W GitHub repo: Settings → Secrets and variables → Actions → Variables
4. Dodaj zmienną: `DISCORD_WEBHOOK_URL` = skopiowany URL
5. Od teraz każdy release wyśle powiadomienie na `#github-feed`

### 4. Opcjonalnie: GitHub native webhook

Dla bogatszych powiadomień (issues, PR, stars):

1. Weź Webhook URL z kroku 3 i dodaj `/github` na końcu
2. W GitHub repo: Settings → Webhooks → Add webhook
3. Payload URL: `https://discord.com/api/webhooks/.../github`
4. Content type: `application/json`
5. Wybierz eventy: Releases, Issues, Pull requests, Stars

## Co skrypt konfiguruje

### 1. Membership Screening (bramka wejściowa)

| Funkcja | Opis |
|---------|------|
| **Community mode** | Włączony automatycznie, #zasady jako kanał regulaminu |
| **Formularz akceptacji** | Nowi członkowie muszą zaakceptować 5 punktów regulaminu |
| **#zasady** | Read-only kanał z pełnym regulaminem serwera |

> Nowy użytkownik widzi popup → klika "Zgadzam się" → dopiero wtedy może pisać.

### 2. Community Onboarding (personalizacja powitalna)

| Pytanie | Opcje |
|---------|-------|
| **Co Cię tu sprowadza?** | Oceniam OpenOMS / Jestem użytkownikiem / Chcę kontrybuować / Przeglądam |
| **Jakie integracje?** | Allegro / InPost & Kurierzy / WooCommerce & Shopify / Wszystko |

Każda odpowiedź automatycznie:
- Przypisuje rolę (@Uzytkownik lub @Contributor)
- Pokazuje odpowiednie kanały

### 3. Server Guide (ekran powitalny)

Strona powitalna z linkami do 4 kluczowych kanałów:
- #general — ogólne rozmowy
- #pomoc-instalacja — pomoc z instalacją
- #contributing — kontrybuowanie
- #bugs — zgłaszanie bugów

### 4. Bezpieczeństwo serwera

| Zabezpieczenie | Opis |
|----------------|------|
| **Verification Level: Medium** | Nowi użytkownicy muszą mieć konto Discord 5+ minut |
| **Content Filter: All Members** | Skanowanie wiadomości WSZYSTKICH użytkowników |
| **2FA dla moderatorów** | Moderatorzy muszą mieć włączone 2FA |
| **Default notifications: Only @mentions** | Brak spamu powiadomień |
| **@everyone hardening** | Zablokowane: @everyone/@here, manage channels/roles/webhooks, kick/ban, create invites |
| **Webhook hardening** | ManageWebhooks tylko dla @Maintainer na wszystkich kanałach |

### 5. AutoMod (6 reguł automatycznej moderacji)

| Reguła | Co robi |
|--------|---------|
| **Blokada spamu** | Filtruje scamy (free nitro, crypto, phishing URLs, NSFW) — blokuje + alert |
| **Mass mention** | Limit 5 wzmianek na wiadomość — blokuje + 5 min timeout |
| **Discord invites** | Blokuje linki zaproszeniowe do innych serwerów |
| **Skracacz URL** | Blokuje bit.ly, tinyurl, t.co itp. (wektor phishingowy) |
| **Pliki wykonywalne** | Blokuje linki do .exe, .bat, .msi, .ps1 itp. |
| **Crypto/web3 spam** | Blokuje presale, whitelist mint, token launch + 10 min timeout |

> Maintainerzy są wykluczeni z AutoMod — mogą pisać wszystko.

### 6. Kanały

| Kategoria | Kanał | Typ | Zabezpieczenia |
|-----------|-------|-----|----------------|
| INFORMACJE | #zasady | Text | Read-only, regulamin |
| | #ogloszenia | Text | Read-only, wiadomość powitalna |
| | #roadmap | Text | Read-only |
| | #changelog | Text | Read-only |
| SPOLECZNOSC | #general | Text | Standardowe |
| | #off-topic | Text | Luźne rozmowy |
| | #pokaz-swoje | Text | Showcase |
| POMOC | #pomoc-instalacja | **Forum** | Tagi: Docker, From Source, PostgreSQL, Solved |
| | #pomoc-konfiguracja | **Forum** | Tagi: SMTP, Auth, Integracje, RBAC, Solved |
| | #pomoc-integracje | **Forum** | Tagi: Allegro, InPost, DHL, WooCommerce, Solved |
| ROZWOJ | #contributing | Text | Standardowe |
| | #pull-requests | Text | Standardowe |
| | #bugs | **Forum** | Tagi: Backend, Frontend, API, Critical, Confirmed, Fixed |
| | #pomysly | **Forum** | Tagi: Under Review, Planned, Implemented, Won't Fix |
| BOTY | #github-feed | Text | Read-only, alerty AutoMod |
| GLOS | #office-hours | Voice | Kanał głosowy na eventy |

### 7. Role

| Rola | Kolor | Specjalne uprawnienia |
|------|-------|----------------------|
| @Maintainer | Czerwony | Pisanie na kanałach read-only, omija AutoMod, zarządzanie webhookami |
| @Contributor | Zielony | Widoczna na liście członków |
| @Uzytkownik | Niebieski | Bazowe uprawnienia |

### 8. Scheduled Events

- **OpenOMS Office Hours** — najbliższa sobota 18:00 CET
- 1 godzina Q&A z maintainerami na kanale głosowym #office-hours
- Język: polski / angielski

## Opcjonalne — do dodania ręcznie

### Carl-bot (darmowy)

Reakcja-role, system sugestii, audit log.

1. Zaproś bota: https://carl.gg
2. Skonfiguruj reaction roles w #zasady
3. Włącz logging (message edits, deletes, joins)

### Answer Overflow (darmowy dla open-source)

Indeksuje forum channele na Google — ludzie mogą znaleźć odpowiedzi bez dołączania do Discorda.

1. Zaproś bota: https://www.answeroverflow.com
2. Wybierz kanały forum do indeksowania

### Cold Owner Account (zalecane)

Przenieś ownership serwera na dedykowane "zimne" konto:

1. Stwórz nowe konto Discord z silnym hasłem + 2FA (klucz sprzętowy)
2. Dołącz do serwera, nadaj rolę admin
3. Server Settings → Members → PPM na konto → Transfer Ownership
4. Wyloguj się z zimnego konta, schowaj credentials w password managerze

Jeśli Twoje główne konto zostanie zhackowane, serwer jest bezpieczny.

## Idempotentność

Skrypt można uruchomić wielokrotnie — nie tworzy duplikatów. Istniejące kanały/role/reguły są pomijane. Brakujące tagi na forach są dodawane.
