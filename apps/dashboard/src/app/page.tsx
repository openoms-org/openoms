"use client";

import { useEffect, useRef } from "react";
import Link from "next/link";
import {
  Package,
  Truck,
  Zap,
  BarChart3,
  Warehouse,
  FileText,
  ScanBarcode,
  TrendingUp,
  Shield,
  ArrowRight,
  Github,
  ChevronRight,
  Check,
  Clock,
  MessageCircle,
} from "lucide-react";

const DISCORD_INVITE = "https://discord.gg/3Z5hzeH5";
const GITHUB_URL = "https://github.com/openoms-org/openoms";

/* ── Intersection Observer fade-in ── */
function useFadeIn() {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const obs = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          el.classList.add("landing-visible");
          obs.unobserve(el);
        }
      },
      { threshold: 0.15 }
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, []);
  return ref;
}

function FadeIn({
  children,
  className = "",
  delay = 0,
}: {
  children: React.ReactNode;
  className?: string;
  delay?: number;
}) {
  const ref = useFadeIn();
  return (
    <div
      ref={ref}
      className={`landing-fade ${className}`}
      style={{ transitionDelay: `${delay}ms` }}
    >
      {children}
    </div>
  );
}

/* ── Features ── */
const FEATURES = [
  {
    icon: Package,
    title: "Zamówienia z jednego miejsca",
    desc: "Allegro, Amazon, WooCommerce, eBay, Kaufland, OLX, Empik — wszystko w jednym dashboardzie. Kanban, filtry, bulk akcje.",
  },
  {
    icon: Truck,
    title: "Etykiety jednym klikiem",
    desc: "InPost, DHL, DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx. Zbiorcze drukowanie, śledzenie, powiadomienia dla klientów.",
  },
  {
    icon: Zap,
    title: "Automatyzacje",
    desc: "Reguły: gdy zamówienie spełnia warunki → zmień status, wyślij email, wygeneruj etykietę, wyślij SMS. Opóźnienia, harmonogramy.",
  },
  {
    icon: TrendingUp,
    title: "Repricing",
    desc: "Automatyczne zarządzanie cenami. Strategie: marża, harmonogram, stan magazynowy. Symulacja przed zastosowaniem.",
  },
  {
    icon: Warehouse,
    title: "Multi-magazyn",
    desc: "Wiele magazynów, dokumenty PZ/WZ/MM, inwentaryzacja, rezerwacje, alerty niskiego stanu. Synchronizacja z marketplace'ami.",
  },
  {
    icon: FileText,
    title: "Faktury i KSeF",
    desc: "Integracja z Fakturownią, inFakt, wFirma. Gotowość na KSeF (Krajowy System e-Faktur). Szablony wydruku.",
  },
  {
    icon: ScanBarcode,
    title: "Stacja pakowania",
    desc: "Skanuj kod kreskowy → zobacz zamówienie → drukuj etykietę. Pick & pack workflow krok po kroku.",
  },
  {
    icon: BarChart3,
    title: "Raporty i analityka",
    desc: "Przychody, trendy, bestsellery, prognoza popytu AI. Eksport CSV. Raporty zaawansowane z filtrami.",
  },
];

/* ── Integrations ── */
const MARKETPLACES = [
  { name: "Allegro", verified: true },
  { name: "Amazon", verified: false },
  { name: "WooCommerce", verified: false },
  { name: "eBay", verified: false },
  { name: "Kaufland", verified: false },
  { name: "OLX", verified: false },
  { name: "Empik / Mirakl", verified: false },
  { name: "Erli", verified: false },
  { name: "Shoper", verified: false },
  { name: "PrestaShop", verified: false },
  { name: "Shopify", verified: false },
];

const CARRIERS = [
  { name: "InPost", verified: true },
  { name: "DHL", verified: false },
  { name: "DPD", verified: false },
  { name: "GLS", verified: false },
  { name: "UPS", verified: false },
  { name: "Poczta Polska", verified: false },
  { name: "Orlen Paczka", verified: false },
  { name: "FedEx", verified: false },
];

/* ── Roadmap ── */
const ROADMAP_DONE = [
  "Integracja Allegro (oferty, zamówienia, Wysyłam z Allegro)",
  "Integracja InPost (paczkomaty, kurier, etykiety, tracking)",
  "Dashboard z kanbanem, ciemnym motywem i PWA",
  "Zarządzanie zamówieniami, produktami, klientami",
  "Silnik automatyzacji (reguły, warunki, akcje)",
  "Stacja pakowania ze skanerem kodów kreskowych",
  "Raporty i eksport CSV",
  "2FA/TOTP, role i uprawnienia",
];

const ROADMAP_IN_PROGRESS = [
  "Kolejne 7 kurierów (DHL, DPD, GLS, UPS, Poczta Polska, Orlen, FedEx)",
  "10 marketplace'ów (Amazon, eBay, WooCommerce, Kaufland, OLX...)",
  "Repricing (4 strategie cenowe)",
  "Multi-magazyn z dokumentami PZ/WZ/MM",
  "Faktury (Fakturownia, inFakt, wFirma) + KSeF",
  "Warianty produktów i bundleki",
  "Wielowalutowość (kursy NBP)",
];

const ROADMAP_NEXT = [
  "Publiczny SaaS (hosted wersja)",
  "Aplikacja mobilna",
  "Marketplace dla wtyczek",
  "Panel klienta (self-service zwroty)",
];

/* ── Page ── */
export default function LandingPage() {
  return (
    <div className="landing dark min-h-screen bg-[#0a0a0b] text-[#e8e8e8] selection:bg-emerald-500/30 selection:text-white">
      {/* ── Noise texture overlay ── */}
      <div className="pointer-events-none fixed inset-0 z-50 opacity-[0.03]" style={{
        backgroundImage: `url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)'/%3E%3C/svg%3E")`,
      }} />

      {/* ── Nav ── */}
      <nav className="fixed top-0 z-40 w-full border-b border-white/5 bg-[#0a0a0b]/80 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-md bg-emerald-500/10 font-mono text-sm font-bold text-emerald-400">
              O
            </div>
            <span className="font-mono text-sm font-semibold tracking-wide text-white">
              OpenOMS
            </span>
          </div>
          <div className="flex items-center gap-4">
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 rounded-lg border border-white/10 px-4 py-2 font-mono text-xs text-[#a0a0a0] transition-all hover:border-white/20 hover:text-white"
            >
              <Github className="h-4 w-4" />
              GitHub
            </a>
            <Link
              href="/login"
              className="rounded-lg bg-emerald-500/10 px-4 py-2 font-mono text-xs font-medium text-emerald-400 transition-all hover:bg-emerald-500/20"
            >
              Zaloguj się
            </Link>
          </div>
        </div>
      </nav>

      {/* ── Hero ── */}
      <section className="relative flex min-h-[85vh] items-center justify-center overflow-hidden pt-16">
        {/* Grid background */}
        <div
          className="pointer-events-none absolute inset-0 opacity-[0.04]"
          style={{
            backgroundImage:
              "linear-gradient(#fff 1px, transparent 1px), linear-gradient(90deg, #fff 1px, transparent 1px)",
            backgroundSize: "64px 64px",
          }}
        />
        {/* Radial glow */}
        <div className="pointer-events-none absolute left-1/2 top-1/3 h-[600px] w-[800px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-emerald-500/5 blur-[120px]" />

        <div className="relative z-10 mx-auto max-w-4xl px-6 text-center">
          <FadeIn>
            <div className="mb-8 inline-flex items-center gap-2 rounded-full border border-emerald-500/20 bg-emerald-500/5 px-4 py-1.5 font-mono text-xs text-emerald-400">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" />
              Open Source &middot; Self-hosted &middot; Bez opłat za zamówienie
            </div>
          </FadeIn>

          <FadeIn delay={100}>
            <h1 className="mb-6 text-4xl font-bold leading-[1.1] tracking-tight text-white sm:text-5xl md:text-6xl lg:text-7xl">
              Zarządzaj zamówieniami
              <br />
              <span className="bg-gradient-to-r from-emerald-400 to-teal-300 bg-clip-text text-transparent">
                bez vendor lock-in
              </span>
            </h1>
          </FadeIn>

          <FadeIn delay={200}>
            <p className="mx-auto mb-10 max-w-2xl text-lg leading-relaxed text-[#808080] sm:text-xl">
              Otwartoźródłowy OMS dla polskich sprzedawców e-commerce.
              <br className="hidden sm:block" />
              Allegro, InPost, automatyzacje, repricing, multi-magazyn —
              wszystko w jednym systemie, pod Twoją kontrolą.
            </p>
          </FadeIn>

          <FadeIn delay={300}>
            <div className="flex flex-col items-center justify-center gap-4 sm:flex-row">
              <a
                href={DISCORD_INVITE}
                target="_blank"
                rel="noopener noreferrer"
                className="group flex items-center gap-3 rounded-xl bg-emerald-500 px-8 py-4 font-semibold text-[#0a0a0b] shadow-lg shadow-emerald-500/20 transition-all hover:bg-emerald-400 hover:shadow-emerald-500/30"
              >
                <MessageCircle className="h-5 w-5" />
                Dołącz na Discord
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
              </a>
              <a
                href={GITHUB_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="group flex items-center gap-3 rounded-xl border border-white/10 bg-white/5 px-8 py-4 font-semibold text-white transition-all hover:border-white/20 hover:bg-white/10"
              >
                <Github className="h-5 w-5" />
                Zobacz na GitHub
                <ChevronRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
              </a>
            </div>
          </FadeIn>

          <FadeIn delay={500}>
            <div className="mt-16 flex items-center justify-center gap-8 font-mono text-xs text-[#555]">
              <span>AGPLv3 + MIT</span>
              <span className="h-3 w-px bg-white/10" />
              <span>Go + Next.js</span>
              <span className="h-3 w-px bg-white/10" />
              <span>PostgreSQL</span>
              <span className="h-3 w-px bg-white/10" />
              <span>Self-hosted lub SaaS</span>
            </div>
          </FadeIn>
        </div>
      </section>

      {/* ── Anti-pitch: 3 columns ── */}
      <section className="relative border-t border-white/5 py-24">
        <div className="mx-auto max-w-6xl px-6">
          <FadeIn>
            <div className="grid gap-px overflow-hidden rounded-2xl border border-white/5 bg-white/5 md:grid-cols-3">
              {[
                {
                  num: "0 zł",
                  label: "za zamówienie",
                  desc: "Żadnych opłat za wolumen. Płacisz tylko za hosting (lub nic, jeśli self-hosted).",
                },
                {
                  num: "100%",
                  label: "Twój kod",
                  desc: "Open source. Forkuj, modyfikuj, dodawaj własne integracje. Żadnych limitów API.",
                },
                {
                  num: "0",
                  label: "vendor lock-in",
                  desc: "Twoje dane na Twoim serwerze. Migracja? Eksportuj i wyjdź w dowolnym momencie.",
                },
              ].map((item) => (
                <div
                  key={item.label}
                  className="bg-[#0f0f10] p-8 md:p-10"
                >
                  <div className="mb-1 font-mono text-4xl font-bold text-emerald-400 md:text-5xl">
                    {item.num}
                  </div>
                  <div className="mb-3 font-mono text-sm uppercase tracking-widest text-[#666]">
                    {item.label}
                  </div>
                  <p className="text-sm leading-relaxed text-[#888]">
                    {item.desc}
                  </p>
                </div>
              ))}
            </div>
          </FadeIn>
        </div>
      </section>

      {/* ── Features ── */}
      <section className="relative border-t border-white/5 py-24">
        <div className="mx-auto max-w-6xl px-6">
          <FadeIn>
            <h2 className="mb-4 text-center font-mono text-xs uppercase tracking-[0.2em] text-emerald-400">
              Funkcje
            </h2>
            <p className="mb-16 text-center text-3xl font-bold text-white sm:text-4xl">
              Wszystko czego potrzebujesz do sprzedaży
            </p>
          </FadeIn>

          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {FEATURES.map((f, i) => (
              <FadeIn key={f.title} delay={i * 60}>
                <div className="group h-full rounded-xl border border-white/5 bg-[#0f0f10] p-6 transition-all hover:border-emerald-500/20 hover:bg-[#111112]">
                  <f.icon className="mb-4 h-6 w-6 text-emerald-400/70 transition-colors group-hover:text-emerald-400" />
                  <h3 className="mb-2 text-sm font-semibold text-white">
                    {f.title}
                  </h3>
                  <p className="text-xs leading-relaxed text-[#777]">
                    {f.desc}
                  </p>
                </div>
              </FadeIn>
            ))}
          </div>
        </div>
      </section>

      {/* ── Integrations ── */}
      <section className="relative border-t border-white/5 py-24">
        <div className="mx-auto max-w-6xl px-6">
          <FadeIn>
            <h2 className="mb-4 text-center font-mono text-xs uppercase tracking-[0.2em] text-emerald-400">
              Integracje
            </h2>
            <p className="mb-16 text-center text-3xl font-bold text-white sm:text-4xl">
              Marketplace&apos;y i kurierzy
            </p>
          </FadeIn>

          <div className="grid gap-12 md:grid-cols-2">
            {/* Marketplaces */}
            <FadeIn>
              <div>
                <h3 className="mb-6 font-mono text-xs uppercase tracking-widest text-[#666]">
                  Marketplace&apos;y
                </h3>
                <div className="space-y-2">
                  {MARKETPLACES.map((m) => (
                    <div
                      key={m.name}
                      className="flex items-center justify-between rounded-lg border border-white/5 bg-[#0f0f10] px-4 py-3"
                    >
                      <span className="text-sm text-[#ccc]">{m.name}</span>
                      {m.verified ? (
                        <span className="flex items-center gap-1 rounded-full bg-emerald-500/10 px-2.5 py-0.5 font-mono text-[10px] font-medium text-emerald-400">
                          <Check className="h-3 w-3" /> Verified
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 rounded-full bg-amber-500/10 px-2.5 py-0.5 font-mono text-[10px] font-medium text-amber-400">
                          <Clock className="h-3 w-3" /> W budowie
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </FadeIn>

            {/* Carriers */}
            <FadeIn delay={100}>
              <div>
                <h3 className="mb-6 font-mono text-xs uppercase tracking-widest text-[#666]">
                  Kurierzy
                </h3>
                <div className="space-y-2">
                  {CARRIERS.map((c) => (
                    <div
                      key={c.name}
                      className="flex items-center justify-between rounded-lg border border-white/5 bg-[#0f0f10] px-4 py-3"
                    >
                      <span className="text-sm text-[#ccc]">{c.name}</span>
                      {c.verified ? (
                        <span className="flex items-center gap-1 rounded-full bg-emerald-500/10 px-2.5 py-0.5 font-mono text-[10px] font-medium text-emerald-400">
                          <Check className="h-3 w-3" /> Verified
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 rounded-full bg-amber-500/10 px-2.5 py-0.5 font-mono text-[10px] font-medium text-amber-400">
                          <Clock className="h-3 w-3" /> W budowie
                        </span>
                      )}
                    </div>
                  ))}
                </div>

                {/* Additional services */}
                <h3 className="mb-6 mt-10 font-mono text-xs uppercase tracking-widest text-[#666]">
                  Inne
                </h3>
                <div className="space-y-2">
                  {[
                    { name: "Fakturownia / inFakt / wFirma", label: "Faktury" },
                    { name: "KSeF", label: "E-faktury" },
                    { name: "SMSAPI / Twilio", label: "SMS" },
                    { name: "OpenAI", label: "AI" },
                  ].map((s) => (
                    <div
                      key={s.name}
                      className="flex items-center justify-between rounded-lg border border-white/5 bg-[#0f0f10] px-4 py-3"
                    >
                      <span className="text-sm text-[#ccc]">{s.name}</span>
                      <span className="rounded-full bg-white/5 px-2.5 py-0.5 font-mono text-[10px] text-[#666]">
                        {s.label}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </FadeIn>
          </div>
        </div>
      </section>

      {/* ── Roadmap ── */}
      <section className="relative border-t border-white/5 py-24">
        <div className="mx-auto max-w-6xl px-6">
          <FadeIn>
            <h2 className="mb-4 text-center font-mono text-xs uppercase tracking-[0.2em] text-emerald-400">
              Roadmap
            </h2>
            <p className="mb-16 text-center text-3xl font-bold text-white sm:text-4xl">
              Co jest gotowe, co planujemy
            </p>
          </FadeIn>

          <div className="grid gap-8 md:grid-cols-3">
            <FadeIn>
              <div className="h-full rounded-xl border border-emerald-500/10 bg-emerald-500/[0.02] p-8">
                <div className="mb-6 flex items-center gap-2">
                  <Check className="h-5 w-5 text-emerald-400" />
                  <h3 className="font-mono text-sm font-semibold uppercase tracking-wider text-emerald-400">
                    Gotowe
                  </h3>
                </div>
                <ul className="space-y-3">
                  {ROADMAP_DONE.map((item) => (
                    <li
                      key={item}
                      className="flex items-start gap-3 text-sm text-[#999]"
                    >
                      <Check className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-500/50" />
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
            </FadeIn>

            <FadeIn delay={100}>
              <div className="h-full rounded-xl border border-cyan-500/10 bg-cyan-500/[0.02] p-8">
                <div className="mb-6 flex items-center gap-2">
                  <Clock className="h-5 w-5 text-cyan-400" />
                  <h3 className="font-mono text-sm font-semibold uppercase tracking-wider text-cyan-400">
                    W trakcie budowy
                  </h3>
                </div>
                <p className="mb-4 text-xs text-[#666]">
                  Kod napisany, wymaga testów produkcyjnych
                </p>
                <ul className="space-y-3">
                  {ROADMAP_IN_PROGRESS.map((item) => (
                    <li
                      key={item}
                      className="flex items-start gap-3 text-sm text-[#999]"
                    >
                      <div className="mt-1.5 h-2 w-2 shrink-0 rounded-full border border-cyan-500/40 bg-cyan-500/20" />
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
            </FadeIn>

            <FadeIn delay={200}>
              <div className="h-full rounded-xl border border-amber-500/10 bg-amber-500/[0.02] p-8">
                <div className="mb-6 flex items-center gap-2">
                  <Clock className="h-5 w-5 text-amber-400" />
                  <h3 className="font-mono text-sm font-semibold uppercase tracking-wider text-amber-400">
                    W planach
                  </h3>
                </div>
                <ul className="space-y-3">
                  {ROADMAP_NEXT.map((item) => (
                    <li
                      key={item}
                      className="flex items-start gap-3 text-sm text-[#999]"
                    >
                      <div className="mt-1.5 h-2 w-2 shrink-0 rounded-full border border-amber-500/40" />
                      {item}
                    </li>
                  ))}
                </ul>
                <div className="mt-8 rounded-lg border border-dashed border-white/10 bg-white/[0.02] p-4 text-center">
                  <p className="text-xs text-[#666]">
                    Masz pomysł na funkcję?{" "}
                    <a
                      href={DISCORD_INVITE}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-emerald-400 hover:underline"
                    >
                      Daj znać na Discordzie
                    </a>
                  </p>
                </div>
              </div>
            </FadeIn>
          </div>
        </div>
      </section>

      {/* ── Beta CTA ── */}
      <section className="relative border-t border-white/5 py-24">
        <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-emerald-500/[0.02] to-transparent" />
        <div className="relative mx-auto max-w-3xl px-6 text-center">
          <FadeIn>
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-emerald-500/20 bg-emerald-500/5 px-4 py-1.5 font-mono text-xs text-emerald-400">
              <Shield className="h-3.5 w-3.5" />
              Szukamy beta-testerów
            </div>
          </FadeIn>

          <FadeIn delay={100}>
            <h2 className="mb-6 text-3xl font-bold text-white sm:text-4xl">
              Sprzedajesz na Allegro?
              <br />
              Przetestuj z nami.
            </h2>
          </FadeIn>

          <FadeIn delay={200}>
            <p className="mb-10 text-lg text-[#808080]">
              Szukamy 10 osób które chcą przetestować OpenOMS na żywym
              organizmie. Nie musisz być techniczny — wystarczy, że masz
              zamówienia i chęć powiedzieć co działa, a co nie.
              <br />
              <span className="text-emerald-400/80">
                Beta-testerzy dostaną dożywotni darmowy plan na SaaS.
              </span>
            </p>
          </FadeIn>

          <FadeIn delay={300}>
            <div className="flex flex-col items-center justify-center gap-4 sm:flex-row">
              <a
                href={DISCORD_INVITE}
                target="_blank"
                rel="noopener noreferrer"
                className="group flex items-center gap-3 rounded-xl bg-emerald-500 px-10 py-4 text-lg font-semibold text-[#0a0a0b] shadow-lg shadow-emerald-500/20 transition-all hover:bg-emerald-400 hover:shadow-emerald-500/30"
              >
                <MessageCircle className="h-5 w-5" />
                Dołącz na Discord
                <ArrowRight className="h-5 w-5 transition-transform group-hover:translate-x-1" />
              </a>
            </div>
          </FadeIn>
        </div>
      </section>

      {/* ── Footer ── */}
      <footer className="border-t border-white/5 py-12">
        <div className="mx-auto max-w-6xl px-6">
          <div className="flex flex-col items-center justify-between gap-6 md:flex-row">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-md bg-emerald-500/10 font-mono text-xs font-bold text-emerald-400">
                O
              </div>
              <span className="font-mono text-sm text-[#555]">
                OpenOMS &copy; {new Date().getFullYear()}
              </span>
            </div>
            <div className="flex items-center gap-6 font-mono text-xs text-[#444]">
              <a
                href={GITHUB_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-[#888]"
              >
                GitHub
              </a>
              <a
                href={DISCORD_INVITE}
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-[#888]"
              >
                Discord
              </a>
              <a
                href={`${GITHUB_URL}/blob/main/CONTRIBUTING.md`}
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-[#888]"
              >
                Contributing
              </a>
              <a
                href={`${GITHUB_URL}/blob/main/SECURITY.md`}
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-[#888]"
              >
                Security
              </a>
              <Link href="/login" className="transition-colors hover:text-[#888]">
                Dashboard
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
