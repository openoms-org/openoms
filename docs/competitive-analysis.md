# Analiza Konkurencji i Best Practices — OpenOMS

## Spis treści

1. [Podsumowanie wykonawcze](#1-podsumowanie-wykonawcze)
2. [BaseLinker — lider rynku](#2-baselinker--lider-rynku)
3. [Polscy konkurenci](#3-polscy-konkurenci)
4. [Platformy międzynarodowe](#4-platformy-międzynarodowe)
5. [Porównanie feature-by-feature](#5-porównanie-feature-by-feature)
6. [Analiza rynku polskiego](#6-analiza-rynku-polskiego)
7. [Trendy branżowe 2025-2026](#7-trendy-branżowe-2025-2026)
8. [UX/UI Best Practices](#8-uxui-best-practices)
9. [Gdzie OpenOMS wypada dobrze](#9-gdzie-openoms-wypada-dobrze)
10. [Gdzie OpenOMS ma luki](#10-gdzie-openoms-ma-luki)
11. [Rekomendacje strategiczne](#11-rekomendacje-strategiczne)

---

## 1. Podsumowanie wykonawcze

### Kontekst rynkowy
- Polski e-commerce: **27.9 mld USD** przychodu (2024), wzrost ~10-15% rocznie
- 86% polskich kupujących korzysta z marketplace'ów
- Allegro dominuje z 60% udziałem, 21M+ użytkowników
- Rynek OMS globalnie: **4.26 mld USD (2025) → 7.46 mld USD (2031)**, CAGR 9.78%

### Kluczowe wnioski
1. **BaseLinker dominuje w Polsce**, ale podwyżki cen (2023) tworzą okno dla alternatyw
2. **Żaden system nie jest kompletny** — każdy ma silne i słabe strony
3. **Brak nowoczesnego, open-source OMS** na polskim rynku = nisza dla OpenOMS
4. **AI, real-time, mobile** — to główne kierunki rozwoju branży
5. **Developer experience** jest słaby u wszystkich konkurentów (brak TypeScript SDK, OpenAPI, playground)

---

## 2. BaseLinker — lider rynku

### Profil
- Rebrand na **Base.com** (2024/2025), nadal powszechnie znany jako BaseLinker
- 1,300+ integracji, 300+ marketplace'ów, ~50 kurierów
- Dziesiątki tysięcy polskich sprzedawców

### Mocne strony

| Obszar | Ocena | Szczegóły |
|--------|-------|-----------|
| Integracje marketplace | ★★★★★ | 300+ marketplace'ów, najgłębsza integracja z Allegro |
| Integracje kurierskie | ★★★★★ | ~50 kurierów, unified API (`createPackage`) |
| Automatyzacja | ★★★★★ | 40+ typów akcji, custom events, if/else, triggery barcode |
| Produkty | ★★★★☆ | Warianty, bundle'e, multi-EAN, auto-SKU, AI background removal |
| Magazyn | ★★★★☆ | Dokumenty, inwentaryzacja, Pick & Pack Assistant, Base WMS |
| Fakturowanie | ★★★★☆ | Wbudowane + integracje (wFirma, Fakturownia, InFakt, iFirma, Xero) |
| Zwroty/RMA | ★★★☆☆ | Portal klienta, 3 scenariusze (zwrot/wymiana/naprawa) |
| Repricing | ★★★★☆ | Allegro, Amazon, eMAG, Ceneo, Empik |

### Słabe strony

| Obszar | Problem |
|--------|---------|
| CRM/Klienci | Order-centric, brak deduplikacji, brak segmentacji |
| UI/UX | Funkcjonalny ale przestarzały, brak dark mode |
| Mobile | Brak aplikacji mobilnej do zarządzania (tylko Android WMS + Caller) |
| API | POST-only, 100 req/min limit, brak natywnych webhooków, brak sandbox |
| Analytics | Basic wbudowane, Base Analytics odświeża dziennie (nie real-time) |
| Raportowanie | Brak report buildera, brak zaplanowanego wysyłania raportów |
| Import | Limit 2MB/plik, 30 importów/dzień |
| Ceny | Nieprzejrzyste Enterprise, obowiązkowy upgrade powyżej 250k USD GMV/mies. |

### Cennik BaseLinker
| Plan | Cena | Limity |
|------|------|--------|
| Freemium | Za darmo | 100 zamówień/mies., 1000 ofert |
| Business | od 99 PLN/mies. + 0.19 PLN/zamówienie | Skaluje się z wolumenem |
| Enterprise | Indywidualnie | >250k USD GMV/mies. lub >5k zamówień/mies. |

### Unikalne funkcje BaseLinker
1. **Base Connect** — sieć B2B łącząca dostawców z sprzedawcami, real-time sync
2. **Repricing** — automatyczne dostosowanie cen na wielu marketplace'ach
3. **AI Listings Agents** — automatyczne tworzenie ofert (Kaufland)
4. **Pick & Pack Assistant** — WMS-lite z barcode scannerem i zdjęciami paczek
5. **Unified Carrier API** — jeden endpoint `createPackage` dla wszystkich kurierów

---

## 3. Polscy konkurenci

### 3.1 Sellasist

**Pozycja:** #2 w czystym OMS w Polsce

| Aspekt | Szczegóły |
|--------|-----------|
| Integracje | 400+, deep Allegro |
| WMS | Dedykowana apka Android z barcode scanning |
| Automatyzacja | 50 reguł automatycznych, webhooki |
| API | REST API, webhook support |
| Cennik | Elastyczny — od darmowego (20 zamówień) do ~149+ PLN/mies. |
| Mocne | Dedykowany WMS mobile, elastyczne ceny, 400+ integracji |
| Słabe | Problemy ze stabilnością, stroma krzywa uczenia |

### 3.2 Apilo (własność Shoper S.A.)

**Pozycja:** #3 i szybko rośnie, backed by cyber_Folks group

| Aspekt | Szczegóły |
|--------|-----------|
| Integracje | 600+, 108 dedykowanych funkcji Allegro |
| WMS | Przez zewnętrzne systemy (NuboWMS, ExpertWMS) |
| Automatyzacja | Rule-based, smart price lists |
| API | REST API na developer.apilo.com |
| Cennik | od **9 PLN/mies.** — najtańszy wejście |
| Mocne | Część grupy Shoper, 600+ integracji, intuicyjny interfejs |
| Słabe | Problemy z integracją hurtowni, sync nie zawsze wystarczająco szybki |

### 3.3 IdoSell (dawniej IAI-Shop)

**Pozycja:** Najwyższy GMV w Polsce (16+ mld PLN), all-in-one platforma

| Aspekt | Szczegóły |
|--------|-----------|
| Typ | Pełna platforma e-commerce (sklep + OMS + WMS + PIM + CRM + POS) |
| WMS | Wbudowany, bez potrzeby zewnętrznego systemu |
| API | API Admin 3 (REST), webhooks, developer portal |
| Cennik | od 29 PLN (Start) do 2,299 PLN (Enterprise)/mies. |
| Mocne | All-in-one, IAI POS, cross-border, wbudowany WMS |
| Słabe | Drogi, stroma krzywa uczenia, nie jest standalone OMS |

### 3.4 Shoper

**Pozycja:** Najpopularniejsza polska platforma SaaS e-commerce (GPW: sWIG80)

| Aspekt | Szczegóły |
|--------|-----------|
| Typ | Platforma e-commerce z OMS przez Apilo |
| Unikalne | Shoper Live (live commerce), OpisyAI (AI opisy produktów) |
| Cennik | od 25 PLN/mies. (Standard) do 5,400 PLN (Enterprise) |
| Mocne | Najlepszy UX wśród polskich platform (4.8/5), publiczna spółka |
| Słabe | Brak dostępu do kodu, obowiązkowa przedpłata roczna |

### 3.5 SOTESHOP (SOTE)

**Pozycja:** Mniejsza, tradycyjna platforma z Poznania

| Aspekt | Szczegóły |
|--------|-----------|
| Integracje | Ograniczone — głównie Allegro, Empik, Ceneo |
| AI | SOTESHOP AI do generowania treści |
| Cennik | od 45 PLN/mies. |
| Mocne | Prostota, dobry support, integracja z Subiekt GT/Nexo |
| Słabe | Ograniczona skalowalność, mało integracji |

### 3.6 Subiekt GT/Nexo (InsERT)

**Pozycja:** Dominuje w ERP/magazyn, wymaga integratorów do e-commerce

| Aspekt | Szczegóły |
|--------|-----------|
| Typ | Desktop ERP/warehouse (nie cloud, nie SaaS) |
| E-commerce | Przez SellIntegro (87+ integracji), GeeShop, BaseLinker |
| API | COM/ActiveX (GT), lepsze w nexo PRO — brak nowoczesnego REST API |
| Cennik | Jednorazowa licencja od 599 PLN + roczne utrzymanie |
| Mocne | Zaufany przez księgowych, pełna zgodność podatkowa PL, batch tracking |
| Słabe | Desktop only, brak REST API, e-commerce wymaga płatnych integratorów |

---

## 4. Platformy międzynarodowe

### Porównanie kluczowych graczy

| Platforma | Cena od | Marketplace'y | WMS | Automatyzacja | API | Mobile | Unique |
|-----------|---------|---------------|-----|---------------|-----|--------|--------|
| **Linnworks** | $449/mies. | 100+ | Add-on (SkuVault) | Dobra | Dobry (SDK) | Nie | Stock forecasting AI |
| **ShipStation** | $9.99/mies. | 400+ | Nie | Silna | Dobry (V2, webhooks) | Tak | Najlepszy UX shippingu |
| **Brightpearl** | ~$99/mies. | Główne | Tak | Doskonała | Doskonały (webhooks QoS) | Nie | Pełny ERP/ROS |
| **Cin7** | $349/mies. | 700+ | Tak (mobile) | Dobra | Dobry | Tak (WMS) | Wbudowane EDI + POS |
| **Veeqo** | **ZA DARMO** | Główne | Tak | Dobra | Dobry (webhooks) | Tak | Amazon rates, darmowy |
| **Rithum** | $2,000+/mies. | 400+ | Nie | AI-driven | Dobry | Nie | Enterprise scale, AI |
| **Ordoro** | $39/mies. | ~20+ | Basic | Basic | Decent | Nie | Flex pricing |
| **Zenventory** | $149/mies. | 70+ | Tak (core) | Basic | Dobry | Tak | 3PL-first |
| **inFlow** | $110/mies. | ~10 | Basic | Basic | Płatny add-on | Tak | Desktop app, Smart Scanner |

### Wyróżniające się rozwiązania

**Brightpearl** — gold standard automatyzacji:
- Automation Engine sprawdza nowe zamówienia **co minutę**
- Pre-built conditions + actions, no-code setup
- Auto-fulfill, auto-invoice, dropshipping, partial fulfillment
- Webhooks z QoS levels i retry policies

**ShipStation** — najlepszy UX shippingu:
- 400+ integracji z kanałami sprzedaży
- AI-driven automation rules dla carrier selection
- Mobilna aplikacja (iOS/Android): label creation, wireless printing
- RSA-SHA256 podpisy na webhookach

**Veeqo** — disruptor cenowy:
- Darmowy (subsydiowany przez Amazon) do kwietnia 2026
- Amazon Buy Shipping rates dla WSZYSTKICH kanałów
- Digital picking z barcode scanning na mobile
- A-to-z Guarantee delivery claims protection

---

## 5. Porównanie feature-by-feature

### Legenda: ✅ Pełne | ⚡ Częściowe | ❌ Brak

| Funkcja | OpenOMS | BaseLinker | Sellasist | Apilo | Brightpearl | ShipStation |
|---------|---------|------------|-----------|-------|-------------|-------------|
| **Zamówienia** |
| CRUD zamówień | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Custom statusy | ✅ | ✅ | ✅ | ✅ | ✅ | ⚡ |
| Merge/split | ✅ | ✅ | ⚡ | ❌ | ✅ | ⚡ |
| Bulk status change | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| CSV import/export | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Custom fields | ✅ | ⚡ | ⚡ | ❌ | ✅ | ⚡ |
| Tagi | ✅ | ⚡ | ⚡ | ❌ | ⚡ | ✅ |
| Inline edit | ✅ | ⚡ | ❌ | ❌ | ❌ | ❌ |
| Kanban view | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Produkty** |
| CRUD produktów | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Warianty | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Bundle'e | ✅ | ✅ | ⚡ | ❌ | ✅ | ❌ |
| Kategorie | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Zdjęcia (upload) | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Waga/wymiary | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| AI background removal | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Repricing | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Magazyn** |
| Multi-warehouse | ✅ | ✅ | ✅ | ✅ | ✅ | ⚡ |
| Dokumenty PZ/WZ/MM | ✅ | ✅ | ⚡ | ⚡ | ✅ | ❌ |
| Barcode scanning | ✅ | ✅ | ✅ (mobile) | ❌ | ⚡ | ❌ |
| Pick & Pack flow | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| Inwentaryzacja | ❌ | ✅ | ⚡ | ❌ | ✅ | ❌ |
| Strict inventory control | ❌ | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Przesyłki** |
| Multi-carrier | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Label generation | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Tracking | ✅ | ✅ (co 4h) | ✅ | ✅ | ✅ | ✅ |
| Paczkomat map | ✅ | ✅ | ⚡ | ⚡ | ❌ | ❌ |
| Rate shopping | ❌ | ❌ | ❌ | ❌ | ⚡ | ✅ |
| Batch label printing | ⚡ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Zwroty** |
| RMA system | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Portal klienta (self-service) | ✅ | ✅ | ⚡ | ❌ | ❌ | ❌ |
| Custom return statuses | ✅ | ✅ | ⚡ | ⚡ | ✅ | ❌ |
| **Klienci** |
| Customer CRUD | ✅ | ⚡ | ✅ | ⚡ | ✅ | ❌ |
| Historia zamówień | ✅ | ✅ | ✅ | ⚡ | ✅ | ⚡ |
| Segmentacja | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |
| Programy lojalnościowe | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Fakturowanie** |
| Wbudowane faktury | ⚡ (Fakturownia) | ✅ | ✅ | ✅ | ✅ | ❌ |
| KSeF ready | ❌ | ⚡ | ⚡ | ⚡ | ❌ | ❌ |
| Korekty | ⚡ | ✅ | ✅ | ✅ | ✅ | ❌ |
| **Automatyzacja** |
| Rules engine | ✅ | ✅ | ✅ | ⚡ | ✅ | ✅ |
| Custom triggers | ✅ | ✅ | ⚡ | ❌ | ✅ | ⚡ |
| If/else logic | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ |
| Action delays | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Visual workflow builder | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Integracje marketplace** |
| Allegro | ✅ (SDK) | ✅ (najgłębsza) | ✅ | ✅ | ❌ | ❌ |
| Amazon | ✅ (SDK) | ✅ | ✅ | ✅ | ✅ | ✅ |
| eBay | ✅ (SDK) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Empik/Erli | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| WooCommerce | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Listing sync | ❌ | ✅ | ✅ | ✅ | ⚡ | ❌ |
| Stock sync real-time | ❌ | ✅ | ✅ | ⚡ | ✅ | ⚡ |
| **Komunikacja** |
| Email notifications | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| SMS notifications | ✅ | ✅ | ⚡ | ⚡ | ❌ | ❌ |
| WhatsApp/Messenger | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **API & Tech** |
| REST API | ✅ | ⚡ (POST-only) | ✅ | ✅ | ✅ | ✅ |
| OpenAPI spec | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ (V2) |
| Webhooks (HMAC signed) | ✅ | ⚡ (via Zapier) | ✅ | ⚡ | ✅ (QoS) | ✅ (RSA-SHA256) |
| WebSocket real-time | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Rate limit | Brak limitu | 100/min | ? | ? | ? | ? |
| **UI/UX** |
| Dark mode | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Mobile responsive | ✅ | ⚡ | ⚡ | ⚡ | ⚡ | ✅ |
| Cmd+K command palette | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Keyboard shortcuts | ✅ | ✅ (barcode) | ❌ | ❌ | ❌ | ❌ |
| PWA | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Zaawansowane** |
| RBAC (granularne role) | ✅ | ⚡ | ⚡ | ⚡ | ⚡ | ⚡ |
| Audit log | ✅ | ❌ | ❌ | ❌ | ⚡ | ❌ |
| Multi-currency | ✅ | ✅ | ⚡ | ⚡ | ✅ | ⚡ |
| B2B pricing | ✅ | ❌ | ❌ | ❌ | ✅ | ❌ |
| Print templates | ✅ | ✅ (HTML/CSS) | ⚡ | ⚡ | ⚡ | ✅ |
| Prometheus monitoring | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Self-hosted option | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Open source | ✅ (AGPL) | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## 6. Analiza rynku polskiego

### Pozycje rynkowe

```
Platformy OMS w Polsce (2025):

  BaseLinker  ████████████████████████████████████  Lider (#1)
  Sellasist   ████████████████                      #2 w czystym OMS
  Apilo       ██████████████                        #3, szybko rośnie
  IdoSell     ████████████████████                  Lider all-in-one
  Shoper      ██████████████████                    Lider SaaS store
  SOTESHOP    ████                                  Nisza
  Subiekt     ██████████████████████                Lider ERP/magazyn
```

### Kluczowe dynamiki rynku

1. **Podwyżki BaseLinker (2023)** → migracja użytkowników do Sellasist i Apilo
2. **Konsolidacja:** Shoper kupił Apilo (100%, 2024), cyber_Folks kupił 49.9% Shoper (2025)
3. **Allegro jest kluczowe** — każdy polski OMS priorytetyzuje integrację z Allegro
4. **KSeF (luty 2026)** — obowiązkowe e-faktury wymuszają upgrade'y
5. **Temu/SHEIN** — presja cenowa na polskich sprzedawców → większe zapotrzebowanie na efektywność

### Czego polscy sprzedawcy potrzebują najbardziej

| # | Potrzeba | Spełnione przez |
|---|----------|-----------------|
| 1 | Stabilność i niezawodność | Żaden system nie jest idealny |
| 2 | Szybka synchronizacja stocku | BaseLinker (najlepsza) |
| 3 | Przystępne ceny | Apilo (od 9 PLN), Veeqo (za darmo) |
| 4 | Prosty interfejs | Shoper (4.8/5), Apilo |
| 5 | Głęboka integracja Allegro | BaseLinker, Apilo (108 funkcji) |
| 6 | Auto label generation | Wszystkie OMS-y |
| 7 | Dostęp mobilny | Prawie nikt (tylko Sellasist WMS Android) |
| 8 | Multi-warehouse | BaseLinker, IdoSell, Sellasist |
| 9 | Obsługa zwrotów | BaseLinker, OpenOMS |
| 10 | Zgodność z KSeF | W przygotowaniu u wszystkich |
| 11 | Narzędzia AI | BaseLinker (AI opisy, category matching) |
| 12 | BLIK/Przelewy24 | Przez platformy sklepowe |

### Powszechne skargi użytkowników

| Problem | Dotknięte platformy |
|---------|---------------------|
| Przestoje/niestabilność | Sellasist, Shoper |
| Wolna synchronizacja | Apilo, IdoSell |
| Skomplikowany cennik | IdoSell, BaseLinker |
| Stroma krzywa uczenia | IdoSell, Sellasist, Subiekt |
| Brak niszowych integracji | Sellasist, SOTESHOP |
| Brak mobile admin | Prawie wszystkie |
| Brak dark mode | Wszystkie |
| Niejasne terminy napraw bugów | Shoper |

---

## 7. Trendy branżowe 2025-2026

### Table stakes (wymagane minimum)

1. Multi-channel order import z głównych marketplace'ów
2. Real-time inventory sync (zapobieganie overselling)
3. Label generation z przynajmniej głównymi kurierami
4. Order status tracking + powiadomienia klienta
5. Podstawowe zarządzanie zwrotami (RMA)
6. Cloud-based deployment
7. Podstawowe dashboardy raportowe
8. REST API access
9. Multi-warehouse inventory visibility

### Differentiatory (przewagi konkurencyjne)

1. **AI demand forecasting** z detekcją sezonowości
2. **Intelligent order routing** — auto-wybór optymalnego magazynu
3. **Agentic AI** — autonomiczna obsługa wyjątków bez ludzkiej interwencji
4. **Real-time WebSocket updates** (nie polling)
5. **Advanced visual workflow builders**
6. **Wbudowane ERP/accounting**
7. **Sustainability tracking** (ślad węglowy na przesyłkę)
8. **Personalized fulfillment** z użyciem danych CRM/loyalty

### AI/ML w nowoczesnych OMS

| Funkcja | Kto ma | Opis |
|---------|--------|------|
| Demand forecasting | Linnworks, Brightpearl | ML models (LSTM, GBM), accuracy do 95% |
| Smart order routing | Brightpearl, Rithum | ML evaluates: stock, proximity, carrier performance |
| Auto-categorization | BaseLinker, Rithum | AI category matching dla marketplace listing |
| AI content generation | BaseLinker, Shoper | AI opisy produktów, multi-language |
| Agentic commerce | Rithum (RithumIQ) | Autonomiczne agenty AI do order routing i exception handling |
| Background removal | BaseLinker | AI usuwanie tła zdjęć produktów |

### Real-time: stan branży

Większość platform OMS nadal używa **polling** (sprawdzanie co 1-60 minut). Prawdziwy real-time (WebSocket/SSE) jest rzadkością.

| Platforma | Webhooks | Real-time dashboard | Push notifications |
|-----------|----------|--------------------|--------------------|
| Brightpearl | Tak (QoS levels) | Near real-time | Via webhooks |
| ShipStation | Tak (RSA podpisy) | Basic | Via webhooks |
| Veeqo | Tak | Near real-time | Mobile app |
| BaseLinker | Via Zapier | Nie | Nie |
| **OpenOMS** | **Tak (HMAC-SHA256)** | **Tak (WebSocket)** | **Tak (PWA)** |

---

## 8. UX/UI Best Practices

### Dashboard

**Must-have KPI (Tier 1 — zawsze widoczne):**
- Total Revenue (dziś/tydzień/miesiąc z trend arrows)
- Orders Today / Pending Orders
- Fulfillment Rate (% zamówień wysłanych na czas)
- Average Order Value (AOV) z porównaniem do poprzedniego okresu

**Tier 2 — wykresy:**
- Revenue Trend (line chart, 7d/30d/90d toggle)
- Order Volume (bar chart by day, overlay z poprzednim okresem)
- Inventory Alerts (low stock / out-of-stock counts)
- Top-selling products (horizontal bar chart)

**Best chart types:**

| Dane | Typ wykresu | Dlaczego |
|------|-------------|----------|
| Revenue over time | Line chart | Trend, comparison overlays |
| Order volume/day | Bar chart | Porównanie dyskretnych okresów |
| Revenue by channel | Donut chart | Proporcja, max 5-6 segmentów |
| Top products | Horizontal bar | Czytelne etykiety, ranking |
| Order status breakdown | Stacked bar / Donut | Kompozycja |

### Nawigacja

**Standard branżowy:** Kolapsowalna lewa boczna nawigacja (sidebar)
- Skaluje się wertykalnie do wielu elementów
- Nie konkuruje z horyzontalną przestrzenią tabel
- Tryb icon-only daje więcej miejsca na content
- Znany pattern (Shopify, Stripe, Linear, Vercel)

**Organizacja 50+ funkcji:**
1. **Grupowanie z collapsible sections** (max 8-10 top-level items)
2. **Progressive disclosure** — zaawansowane ukryte za expandable sections
3. **Command palette (Cmd+K)** — escape hatch dla power userów

**Role-based navigation:**
- Admin: widzi wszystko
- Operations Manager: Zamówienia, Fulfillment, Inventory, Reports
- Warehouse Staff: Picking Queue, Packing, Shipping only
- Customer Service: Orders (read-only), Customers, Returns
- Finance: Reports, Analytics, Billing

### Tabele

**Kolumny:**
- Show/hide columns z zapamiętywaniem preferencji
- Drag-to-resize na column borders
- Density toggle (compact/default/comfortable)
- Saved views (nazwane konfiguracje kolumn)

**Inline editing:**
- Click cell to edit → Enter to save → Escape to cancel
- Dla prostych pól (ilość, status, notatki)
- Dla złożonych edycji — slide-over panel (prawa strona)

**Bulk actions:**
- Checkbox column (leftmost)
- Floating action bar pojawia się po zaznaczeniu 1+ rows
- "Select all on this page" + "Select all X matching results"

**Filtrowanie (3 warstwy):**
1. Quick status tabs + search + date range (zawsze widoczne)
2. Advanced filter builder ("Add filter" → pill: field + operator + value)
3. Saved filter presets (named views)

**Paginacja:** Tradycyjna (25/50/100 per page) — standard dla admin paneli. NIE infinite scroll.

### Order Detail Page — layout

```
+--------------------------------------------------+
| ← Zamówienia    #ORD-1234              [Akcje ▼]  |
+--------------------------------------------------+
|                                                    |
| LEWA KOLUMNA (60-65%)  | PRAWA KOLUMNA (35-40%)   |
|                        |                           |
| [Status Banner]        | [Klient]                  |
| Status + payment       | Imię, email, telefon      |
|                        | Klient od: data           |
| [Timeline]             |                           |
| Historia statusów      | [Adres dostawy]           |
| z timestamps           | Pełny adres, kopiowanie   |
|                        |                           |
| [Pozycje zamówienia]   | [Adres rozliczeniowy]     |
| Produkt, SKU, ilość    |                           |
| cena, suma             | [Podsumowanie płatności]  |
|                        | Subtotal, podatek, wysyłka|
| [Fulfillment]          | rabat, total              |
| Tracking, kurier       |                           |
| Etykieta               | [Tagi / Notatki]          |
|                        |                           |
| [Activity Log]         |                           |
+--------------------------------------------------+
```

### Zmiana statusu — UX

| Podejście | Kiedy |
|-----------|-------|
| Action buttons ("Wyślij", "Anuluj") | Na stronie szczegółów — najlepsze |
| Kanban board | Przegląd wszystkich zamówień, drag-and-drop |
| Timeline/Stepper | Read-only display postępu |
| Dropdown | Prosty linear workflow |

### Dark Mode

- **3 opcje:** Light / Dark / System preference
- Default: System preference
- Implementacja: CSS custom properties / Tailwind `dark:` + `next-themes`
- Testować wszystkie wykresy i statusy w obu trybach

### Command Palette (Cmd+K) — standard 2025

Powinno szukać:
- Nawigacja (Przejdź do Zamówień, Produktów)
- Encje (zamówienia po numerze, klienci po nazwie)
- Akcje (Utwórz zamówienie, Eksportuj raport, Drukuj etykiety)
- Ustawienia (Toggle dark mode, Zmień język)
- Ostatnie elementy (5-10 ostatnio wyświetlonych)

### Powiadomienia

| Typ | Use case | Zachowanie |
|-----|----------|------------|
| Toast | Feedback na akcje użytkownika | Auto-dismiss 3-5s, top-right, non-blocking |
| Notification Center | Async events (nowe zamówienie, low stock) | Persistent, bell icon, badge count |

### Loading

- **Skeleton screens** — dla ładowania stron i tabel (odczuwane jako szybsze)
- **Spinner (mały)** — inline actions (saving field, loading dropdown)
- **Progress bar** — file uploads, exports, batch operations
- **Optimistic updates** — status changes, edycje (UI natychmiast, potem reconcile z serwerem)

### Wzorce z najlepszych admin paneli

| Panel | Key Pattern |
|-------|------------|
| **Shopify** | Card-based layout, contextual actions, progressive disclosure |
| **Stripe** | Information density done right, tabular-nums, test mode toggle |
| **Linear** | Speed-first (3.7x szybszy niż Jira), keyboard-first, real-time sync |
| **Vercel** | Typography discipline, status without color dependency, resilient layouts |

**10 wspólnych cech najlepszych paneli:**
1. Szybkość jest #1 feature (optimistic updates, SWR, skeleton loading)
2. Opinionated defaults z progressive customization
3. Keyboard-first, ale mouse-friendly
4. Spójny layout across all pages
5. Clear information hierarchy (typography size, weight, spacing)
6. Non-blocking interactions (toasts, auto-save, optimistic updates)
7. Excellent empty states i error handling
8. Accessibility as foundation (Radix UI primitives)
9. Cmd+K as universal search/navigation
10. Design system, nie ad-hoc components

---

## 9. Gdzie OpenOMS wypada dobrze

### Przewagi nad konkurencją

| Przewaga | Szczegóły | Kto nie ma |
|----------|-----------|------------|
| **Open source (AGPL)** | Jedyny open-source OMS dla PL e-commerce | Wszyscy konkurenci |
| **Self-hosted** | Pełna kontrola nad danymi | Wszyscy SaaS |
| **WebSocket real-time** | Live dashboard updates, nie polling | BaseLinker, Sellasist, Apilo |
| **Dark mode** | Light/Dark/System toggle | Żaden konkurent |
| **Cmd+K palette** | Szybka nawigacja, search, akcje | Żaden konkurent |
| **Keyboard shortcuts** | Pełny system z skrótami | Tylko BaseLinker (barcode) |
| **PWA** | Offline-capable, installable | Żaden konkurent |
| **OpenAPI 3.1 spec** | Dokumentacja API formalna | Tylko ShipStation (V2) |
| **Webhooks HMAC-SHA256** | Signed, verified, delivery log | BaseLinker brak natywnych |
| **RBAC granularne** | Custom roles z permissions matrix | Konkurenci mają basic |
| **Audit log** | Pełny log zmian z filtrami | Prawie nikt |
| **Multi-tenant RLS** | Izolacja na poziomie bazy | Nie dotyczy (SaaS) |
| **Prometheus metrics** | Monitoring ready | Żaden konkurent |
| **Automation rules engine** | Trigger → conditions → actions | Porównywalny z BaseLinker |
| **B2B pricing (cenniki)** | Dedykowane cenniki B2B | Tylko Brightpearl |
| **Returns self-service portal** | Publiczny formularz zwrotu | Tylko BaseLinker |
| **Nowoczesny stack** | Go + Next.js 16 + React 19 + shadcn/ui | Konkurenci mają starsze UI |

### Porównanie techniczne z BaseLinker

| Aspekt | OpenOMS | BaseLinker |
|--------|---------|------------|
| API style | REST (GET/PUT/POST/DELETE) | POST-only |
| API spec | OpenAPI 3.1 | Brak |
| Rate limit | Brak | 100/min |
| Webhooks | Native, HMAC-SHA256, delivery log | Via Zapier (3 event types) |
| Real-time | WebSocket | Brak |
| Auth | Ed25519 JWT | Token-based |
| Multi-tenant | PostgreSQL RLS | N/A (SaaS) |
| Deploy | Self-hosted / Docker | SaaS only |
| License | AGPL v3 | Proprietary |
| Frontend | Next.js 16 + shadcn/ui | Custom (dated) |

---

## 10. Gdzie OpenOMS ma luki

### Krytyczne luki (MUST HAVE dla konkurencyjności)

| # | Luka | BaseLinker | Wpływ |
|---|------|------------|-------|
| 1 | **Listing sync z marketplace'ami** | ✅ Pełna synchronizacja ofert | Bez tego sprzedawcy muszą ręcznie zarządzać ofertami |
| 2 | **Real-time stock sync** | ✅ Auto-sync cross-channel | Ryzyko oversellingu bez tego |
| 3 | **Inwentaryzacja (stocktaking)** | ✅ Pełna z barcode | Kluczowe dla warehouse operations |
| 4 | **KSeF compliance** | ⚡ W przygotowaniu | Obowiązkowe od 02/2026 |
| 5 | **Pełne fakturowanie wbudowane** | ✅ Wbudowane | Obecnie tylko integracja z Fakturownia |

### Ważne luki (SHOULD HAVE)

| # | Luka | BaseLinker | Wpływ |
|---|------|------------|-------|
| 6 | **Repricing** | ✅ Multi-marketplace | Duży differentiator BaseLinker |
| 7 | **Base Connect (B2B network)** | ✅ Dropshipping network | Network effect |
| 8 | **AI content generation** | ✅ Opisy, parametry | Productivity boost |
| 9 | **AI category matching** | ✅ Allegro, eMAG, Erli | Szybsze listowanie |
| 10 | **Pick & Pack z photo verification** | ✅ Zdjęcia paczek | QC dla warehouse |
| 11 | **Strict inventory control** | ✅ Blokada manualnych zmian | Kluczowe dla dużych magazynów |
| 12 | **Kanban board dla zamówień** | ❌ (żaden) | UX differentiator |
| 13 | **Action delays w automatyzacji** | ✅ Opóźnione akcje | Ważne dla workflow timing |
| 14 | **Rate shopping (porównanie stawek kurierów)** | ❌ | ShipStation ma, oszczędność $$ |

### Nice-to-have luki

| # | Luka | Kto ma | Notatki |
|---|------|--------|---------|
| 15 | Visual workflow builder | Nikt (Zapier-style) | Duży differentiator UX |
| 16 | Demand forecasting AI | Linnworks, Brightpearl | ML-based predictions |
| 17 | Customer segmentation | Brightpearl | CRM feature |
| 18 | Live commerce (Shoper Live) | Shoper | Niszowe, ale trendy |
| 19 | EDI support | Cin7 | B2B/wholesale |
| 20 | Widget-based customizable dashboard | Nikt (wszyscy fixed) | UX differentiator |

---

## 11. Rekomendacje strategiczne

### Pozycjonowanie OpenOMS

```
                    COMPLEXITY
                    ▲
                    │
   Rithum          │          Brightpearl
   (enterprise)    │          (full ERP)
                    │
                    │     IdoSell
                    │     (all-in-one)
                    │
    Subiekt         │         BaseLinker
    (ERP/desktop)   │         (market leader)
                    │
                    │    Sellasist    ← OpenOMS target zone
                    │    Apilo           (modern, open-source,
                    │                     developer-friendly)
                    │
   SOTESHOP        │    ShipStation
   (simple)        │    (shipping-first)
                    │
                    └──────────────────────► FEATURES
```

### TOP 10 priorytetów rozwoju

| # | Priorytet | Dlaczego | Effort |
|---|-----------|----------|--------|
| 1 | **Real-time stock sync z marketplace** | Bez tego system nie jest production-ready | XL |
| 2 | **Listing sync (Allegro first)** | Core value prop dla polskich sprzedawców | XL |
| 3 | **KSeF e-faktury** | Obowiązkowe od 02/2026 | L |
| 4 | **Inwentaryzacja/Stocktaking** | Brak = niekompletny WMS | M |
| 5 | **Kanban board zamówień** | Żaden konkurent nie ma, UX differentiator | M |
| 6 | **Action delays w automatyzacji** | Ważne dla realnych workflow'ów | S |
| 7 | **AI opisy produktów** | AI jest teraz expected, nie bonus | M |
| 8 | **Visual workflow builder** | Nikt nie ma → unicorn feature | L |
| 9 | **Rate shopping (carrier prices)** | Bezpośredni wpływ na koszty sprzedawcy | M |
| 10 | **Strict inventory control mode** | Must-have dla profesjonalnych magazynów | M |

### Strategia różnicowania

**OpenOMS powinien pozycjonować się jako:**

> "Jedyny open-source, self-hosted OMS dla polskiego e-commerce z nowoczesnym UX (dark mode, Cmd+K, real-time), silnym API (OpenAPI 3.1, WebSocket, signed webhooks), i pełną kontrolą nad danymi."

**3 główne przewagi do komunikowania:**
1. **Open source + self-hosted** — pełna kontrola, brak vendor lock-in, brak limitu zamówień
2. **Modern developer experience** — OpenAPI spec, TypeScript SDK, WebSocket, HMAC webhooks
3. **Modern UX** — dark mode, Cmd+K, PWA, keyboard shortcuts, real-time — czego żaden konkurent nie oferuje

---

*Raport wygenerowany: 2025-02-09*
*Źródła: Web research 4 agentów (BaseLinker deep dive, polscy konkurenci, międzynarodowe OMS, UX best practices)*
