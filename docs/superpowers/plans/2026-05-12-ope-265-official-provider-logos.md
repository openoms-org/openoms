# OPE-265 Official Provider Logos Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add official provider logo support without exposing unapproved or unverified provider brands.

**Architecture:** Keep provider identity centralized in `ProviderInfo` and `ProviderLogo`. Add an optional official asset path to provider brand metadata; `ProviderLogo` renders that asset when present and falls back to the existing wordmark/initials path otherwise.

**Tech Stack:** Next.js 16, React 19, TypeScript, Tailwind v4, Vitest, Testing Library.

---

### Task 1: Official Asset Metadata

**Files:**
- Modify: `apps/dashboard/src/lib/provider-info.ts`
- Test: `apps/dashboard/src/components/shared/__tests__/provider-identity.test.tsx`

- [ ] **Step 1: Write failing tests**

Update the provider identity test so InPost must expose an official image asset while Allegro remains on the safe fallback:

```tsx
expect(screen.getByRole("img", { name: "InPost logo" })).toHaveAttribute(
  "src",
  "/logos/official/inpost.svg",
);
expect(screen.getByLabelText("Allegro logo")).toHaveTextContent("Allegro");
```

- [ ] **Step 2: Verify RED**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/provider-identity.test.tsx --reporter=dot
```

Expected: fails because `ProviderLogo` currently renders InPost as a text mark, not an image asset.

- [ ] **Step 3: Add brand asset metadata**

Extend `ProviderBrand` with:

```ts
officialAsset?: {
  src: string;
  source: string;
  reviewedAt: string;
};
```

Set InPost only:

```ts
officialAsset: {
  src: "/logos/official/inpost.svg",
  source: "https://inpost.pl/do-pobrania",
  reviewedAt: "2026-05-12",
}
```

- [ ] **Step 4: Verify metadata compiles**

Run the same targeted Vitest command. Expected: still fails until rendering support is added.

### Task 2: Official Logo Rendering

**Files:**
- Modify: `apps/dashboard/src/components/shared/provider-logo.tsx`
- Create: `apps/dashboard/public/logos/official/inpost.svg`
- Create: `apps/dashboard/public/logos/official/README.md`

- [ ] **Step 1: Add the official InPost asset**

Download or copy the official InPost SVG from `https://inpost.pl/do-pobrania` into:

```text
apps/dashboard/public/logos/official/inpost.svg
```

Document the source in:

```text
apps/dashboard/public/logos/official/README.md
```

- [ ] **Step 2: Render official asset when present**

In `ProviderLogo`, when `brand.officialAsset` exists, render an `img` inside the existing mark container:

```tsx
<img
  src={brand.officialAsset.src}
  alt={`${name} logo`}
  className="h-full max-h-6 w-auto object-contain"
/>
```

Keep `data-provider-key` on the wrapper and keep fallback rendering unchanged for providers without an official asset.

- [ ] **Step 3: Verify GREEN**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/provider-identity.test.tsx --reporter=dot
```

Expected: provider identity tests pass.

### Task 3: Validation And PR

**Files:**
- Modify: `apps/dashboard/src/middleware.ts`
- Test: `apps/dashboard/src/__tests__/middleware.test.ts`
- Modify: `docs/superpowers/specs/2026-05-12-ope-260-dashboard-ui-system.md`

- [ ] **Step 1: Keep logo assets publicly reachable**

Add a middleware regression test that `/logos/official/inpost.svg` is not redirected to `/login`, while a protected page such as `/carriers` still is. Then allow `/logos/` through `isPublicPath` and the middleware matcher.

- [ ] **Step 2: Update UI system spec**

Append a short note that official provider logos may be used only when sourced from official/approved assets, with safe wordmark fallback otherwise.

- [ ] **Step 3: Run focused checks**

Run:

```bash
cd apps/dashboard
npx vitest run src/__tests__/middleware.test.ts src/components/shared/__tests__/provider-identity.test.tsx --reporter=dot
npm run lint:quiet
```

Expected: both pass.

- [ ] **Step 4: Run repo checks before push**

Run:

```bash
cd ../..
git diff --check
./scripts/local-ci.sh
```

Expected: no whitespace errors and full local CI passes.

- [ ] **Step 5: Self-review**

Run:

```bash
git diff --stat
git diff
```

Confirm no new unverified provider is exposed and no unapproved logo is introduced.
