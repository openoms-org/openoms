# OpenOMS Discord Server Setup

Jednorazowy skrypt konfigurujący serwer Discord z kanałami, rolami i uprawnieniami.

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

## Co skrypt tworzy

### Role
| Rola | Kolor | Opis |
|------|-------|------|
| @Maintainer | Czerwony | Core team — mogą pisać na kanałach read-only |
| @Contributor | Zielony | Kontrybutorzy kodu |
| @Uzytkownik | Niebieski | Użytkownicy |

### Kanały
| Kategoria | Kanał | Opis |
|-----------|-------|------|
| INFORMACJE | #ogloszenia | Read-only, wiadomość powitalna |
| | #roadmap | Read-only, link do ROADMAP.md |
| | #changelog | Read-only, logi zmian |
| SPOLECZNOSC | #general | Ogólne rozmowy |
| | #pokaz-swoje | Showcase instalacji |
| | #pomysly | Propozycje funkcji |
| POMOC | #instalacja | Setup & deployment |
| | #konfiguracja | Settings & config |
| | #integracje | Allegro, InPost, marketplace'y |
| ROZWOJ | #contributing | Architektura, code review |
| | #bugs | Dyskusja o bugach |
| | #pull-requests | PR discussions |
| BOTY | #github-feed | Read-only, auto-notyfikacje |

Skrypt jest idempotentny — ponowne uruchomienie nie tworzy duplikatów.
