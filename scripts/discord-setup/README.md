# OpenOMS Discord Server Setup

Jednorazowy skrypt konfigurujący serwer Discord z kanałami, rolami, uprawnieniami i politykami bezpieczeństwa.

## Wymagania

- Node.js 18+
- Bot Discord z uprawnieniami Administratora

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

### Membership Screening (bramka wejściowa)

| Funkcja | Opis |
|---------|------|
| **Community mode** | Włączony automatycznie, #zasady jako kanał regulaminu |
| **Formularz akceptacji** | Nowi członkowie muszą zaakceptować 5 punktów regulaminu zanim uzyskają dostęp |
| **#zasady** | Read-only kanał z pełnym regulaminem serwera |

> Nowy użytkownik widzi popup → klika "Zgadzam się" → dopiero wtedy może pisać. Boty/spamerzy tego nie robią.

### Bezpieczeństwo serwera

| Zabezpieczenie | Opis |
|----------------|------|
| **Verification Level: Medium** | Nowi użytkownicy muszą mieć konto Discord 5+ minut |
| **Content Filter: All Members** | Skanowanie wiadomości WSZYSTKICH użytkowników (nie tylko bez roli) |
| **2FA dla moderatorów** | Moderatorzy muszą mieć włączone 2FA żeby banować/kickać/zarządzać |
| **Default notifications: Only @mentions** | Nowi członkowie nie dostają powiadomień z każdej wiadomości |
| **@everyone hardening** | Zablokowane: @everyone/@here, manage channels/roles/webhooks, kick/ban, create invites |

### AutoMod (automatyczna moderacja)

| Reguła | Co robi |
|--------|---------|
| **Blokada spamu** | Filtruje scamy (free nitro, crypto, phishing URLs, NSFW) — blokuje + alert |
| **Mass mention** | Limit 5 wzmianek na wiadomość — blokuje + 5 min timeout |
| **Discord invites** | Blokuje linki zaproszeniowe do innych serwerów |

> Maintainerzy są wykluczeni z AutoMod — mogą pisać wszystko.

### Kanały

| Kategoria | Kanał | Zabezpieczenia |
|-----------|-------|----------------|
| INFORMACJE | #zasady | Read-only, regulamin serwera |
| | #ogloszenia | Read-only, wiadomość powitalna |
| | #roadmap | Read-only |
| | #changelog | Read-only |
| SPOLECZNOSC | #general | Standardowe |
| | #pokaz-swoje | Standardowe |
| | #pomysly | Standardowe |
| POMOC | #instalacja | Slow mode 10s |
| | #konfiguracja | Slow mode 10s |
| | #integracje | Slow mode 10s |
| ROZWOJ | #contributing | Standardowe |
| | #bugs | Standardowe |
| | #pull-requests | Standardowe |
| BOTY | #github-feed | Read-only, alerty AutoMod |

### Role

| Rola | Kolor | Specjalne uprawnienia |
|------|-------|----------------------|
| @Maintainer | Czerwony | Pisanie na kanałach read-only, omija AutoMod |
| @Contributor | Zielony | Widoczna na liście członków |
| @Uzytkownik | Niebieski | Bazowe uprawnienia |

## Idempotentność

Skrypt można uruchomić wielokrotnie — nie tworzy duplikatów. Istniejące kanały/role/reguły są pomijane.
