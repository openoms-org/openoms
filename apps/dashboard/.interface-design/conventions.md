# Interaction & Code Conventions

Behavioral conventions for the dashboard (loading state, buttons, toasts). The
visual design language (color, depth, typography, spacing, layout) lives in
[`system.md`](./system.md); this file is the orthogonal, code-facing companion.

These rules are partially enforced automatically — see [Enforcement](#enforcement).

## 1. React Query state naming

The flag name must match the React Query property the value originates from:

- **Mutations** (`useMutation`) expose **`isPending`** — use it for "request in
  flight".
- **Queries** (`useQuery`) expose **`isLoading`** — use it for first-load state.
- Non-React-Query loading (e.g. the Zustand auth-hydration flag in
  `lib/auth.ts` / `use-auth.ts`) legitimately uses `isLoading`.

A shared component prop that carries a **mutation's** in-flight state MUST be
named `isPending`, not `isLoading`. Call sites already pass `mutation.isPending`;
naming the prop `isLoading` only hides the mutation semantics and is the defect.
A prop that carries a **query's** loading state (e.g. dashboard `stat-cards`,
`orchestration-map` fed by `useOperationsDashboard`) correctly keeps `isLoading`.

```tsx
// ✓ mutation in-flight prop
interface ActionDialogProps { isPending?: boolean; }
<ActionDialog isPending={deleteOrder.isPending} />

// ✗ same value, wrong name
interface ActionDialogProps { isLoading?: boolean; }
<ActionDialog isLoading={deleteOrder.isPending} />
```

## 2. Buttons that trigger a mutation

Every action/submit button wired to a mutation MUST:

1. be `disabled` while the mutation is pending (combined with form validity /
   confirm-disabled where relevant): `disabled={m.isPending}`;
2. show a **localized** loading affordance while pending — a spinner and/or a
   `t()` loading label. Disabling alone, with no visible feedback, is the worst
   pattern and is not allowed.

```tsx
<Button disabled={m.isPending || !isValid}>
  {m.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
  {m.isPending ? t("common.processing") : t("save")}
</Button>
```

- Loading text MUST come from `t()` (next-intl). Hardcoded loading strings
  (`"Zapisywanie..."`, `"Tworzenie..."`, `"Zatwierdzanie..."`, …) are forbidden.
  Reuse `common.processing` / `common.loading` / `common.saving` for generic
  actions, or an action-specific key where one already exists.
- Icon-only buttons MUST animate the icon (`animate-spin`) while pending.
- For status-transition button **groups** (order / shipment / return / dropship
  transitions): disable the whole group while pending and show a single visible
  in-flight signal (a spinner on the clicked button or a group-level spinner).
- The shared `ActionDialog`, `ConfirmDialog`, and `StatusTransitionDialog`
  wrappers already implement this correctly (spinner + `t("processing")`) —
  prefer them over ad-hoc buttons.

## 3. Form submit (`FormWrapper`)

A `FormWrapper` driving a mutation MUST pass a localized `submittingLabel`
(default `t("common.processing")`) so the submit button reflects in-flight
state instead of staying on its static label.

## 4. Toasts

Toast library: [`sonner`](https://sonner.emilkowal.ski/), imported directly as
`import { toast } from "sonner"` (the `<Toaster />` is mounted in
`app/layout.tsx`). API: `toast.success` / `toast.error` / `toast.warning` /
`toast.info`.

- **Firing.** Success/info toasts fire only **after** the request settles. Two
  styles are both compliant — keep whichever a file already uses:
  - inside the mutation's `onSuccess` / `onError` callbacks (preferred), or
  - after `await mutation.mutateAsync(...)` in a `try` block, with the error
    toast in `catch`.

  Pre-await / optimistic success toasts are forbidden. A validation-guard toast
  fired before `mutate()` is allowed (it is not a result toast) but its text
  must still be localized.
- **Localization (the rule most often broken).** Every toast message MUST be a
  `t()` call. Raw string literals and template literals — including ones that
  interpolate server data (`` `Zlecenie odbioru #${data.id}` ``) — are
  forbidden; use `t()` with interpolation params.
- **Error extraction.** Error toasts MUST use `getErrorMessage(err)` from
  `@/lib/api-client`. The inline `err instanceof Error ? err.message : "<Polish>"`
  fallback is not allowed — at minimum use a `t()` key.
- **Completeness.** Every state-changing mutation MUST give the user both
  success and error feedback. A silent `mutate` / `mutateAsync` is a defect.

## Enforcement

- **ESLint** (`no-restricted-syntax` in `eslint.config.mjs`) flags hardcoded
  **string-literal** toast arguments (`toast.success("…")`). Currently at `warn`;
  promote to `error` once all existing violations are cleared.
- **Vitest regression test** (`src/lib/__tests__/i18n-hardcoded-copy.test.ts`)
  statically scans audited files for forbidden hardcoded Polish copy — this
  catches the toast/button **template literals** ESLint cannot, and is wired
  into CI. Add a file to `auditedFiles` and its phrases to `forbiddenPhrases`
  as you localize it.
- The `isPending`/`isLoading` naming and "button must show loading text" rules
  are not cleanly machine-enforceable (they need type/dataflow context); they
  rely on review against this document.
