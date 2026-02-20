---
paths:
  - "apps/dashboard/**/*.ts"
  - "apps/dashboard/**/*.tsx"
---

# React/Next.js Conventions — OpenOMS Dashboard

## Component Patterns
- Functional components only (no class components)
- shadcn/ui primitives from `components/ui/` — don't reinvent buttons, dialogs, tables
- Business components: `components/shared/` (reusable) or domain dirs (`components/orders/`)
- Page components: `app/(dashboard)/xxx/page.tsx` (Next.js App Router)

## Data Fetching
- React Query for all server state — custom hooks in `hooks/use-xxx.ts`
- `useQuery` for reads, `useMutation` for writes
- Always invalidate related queries on mutation success
- API client: `lib/api-client.ts` — NEVER use raw `fetch()` for API calls

## State Management
- Server state: React Query (tanstack)
- Auth state: Zustand store (`lib/auth.ts`)
- Form state: React Hook Form + Zod v4 resolver
- No Redux, no Context for server data

## Styling
- Tailwind v4 classes — no CSS modules, no styled-components
- `cn()` utility for conditional classes: `cn("base-class", condition && "conditional-class")`
- Responsive: mobile-first (`sm:`, `md:`, `lg:` breakpoints)

## Types
- All API types in `types/api.ts` (single source of truth, 306+ interfaces)
- Use strict TypeScript — no `any` unless absolutely necessary
- Zod schemas for form validation (derive types with `z.infer<typeof schema>`)

## Localization
- UI labels in Polish: `Zamowienia`, `Produkty`, `Ustawienia`, `Zapisz`, `Anuluj`, etc.
- Code/comments in English
- Date format: `dd.MM.yyyy` (Polish standard)
- Currency: PLN default, support multi-currency via exchange rates

## Testing
- Unit tests: Vitest + Testing Library (`__tests__/`)
- E2E tests: Playwright (`e2e/`)
- E2E auth helper: `gotoWithAuth(page, path)`
- E2E selectors: Polish labels (`getByText("Zamowienia")`)

## Toast Notifications
- Success: `toast.success("Zapisano pomyslnie")`
- Error: `toast.error(error.message || "Wystapil blad")`
- Use `sonner` library (already configured)
