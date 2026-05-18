# OPE-396 Login Constellation Access Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the dashboard login and 2FA screens to match the accepted Constellation Access direction while preserving all existing auth behavior.

**Architecture:** Keep the change route-local to the auth login page. The login page owns the decorative background, brand lockup, card shell, form content, and 2FA state so other auth pages are not accidentally restyled. Use the existing auth hooks, shadcn primitives, i18n messages, and form validation.

**Tech Stack:** Next.js App Router, React 19, TypeScript, Tailwind v4, shadcn/ui, react-hook-form, zod, next-intl, Vitest, Testing Library.

---

## Scope And Boundaries

This PR only changes the public dashboard login experience for OPE-396.

Do:
- implement the accepted Constellation Access layout,
- use the OpenOMS brand mark from `brand-assets/community-logo/openoms-community-logo-v2-slack-512.png`,
- preserve tenant slug, email, password, submit/loading, error toast, password visibility toggle, conditional registration link, and 2FA flow,
- add `prefers-reduced-motion` support for decorative animation,
- add tests for login rendering, password toggle behavior, and 2FA transition rendering,
- validate visually in the browser on desktop and mobile.

Do not:
- add a language switcher,
- add feature tiles, bottom explainer cards, fake status widgets, or other non-functional interactions,
- change backend auth APIs,
- change registration or billing screens,
- change dashboard layout.

## File Structure

- Modify: `apps/dashboard/src/app/(auth)/login/page.tsx`
  - Replace the old `Package`/`Card` presentation with route-local Constellation Access components.
  - Keep `useAuth`, `usePublicConfig`, `react-hook-form`, zod validation, and 2FA behavior.
- Create: `apps/dashboard/src/app/(auth)/login/page.test.tsx`
  - Cover route-local presentation and auth behavior without requiring a browser.
- Modify: `apps/dashboard/messages/pl/layout.json`
  - Update login title/subtitle and add the security note copy.
- Modify: `apps/dashboard/messages/en/layout.json`
  - Keep English messages complete for i18n parity.
- Copy asset: `apps/dashboard/public/logos/openoms-community-logo-v2-slack-512.png`
  - Static logo used by the login page.

## Task 1: Add Login Tests First

**Files:**
- Create: `apps/dashboard/src/app/(auth)/login/page.test.tsx`

- [x] **Step 1: Write the failing tests**

Create `apps/dashboard/src/app/(auth)/login/page.test.tsx` with mocked auth/config/i18n modules and assertions for the accepted UI.

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import LoginPage from "./page";

const loginMock = vi.fn();
const verify2FALoginMock = vi.fn();

vi.mock("next/link", () => ({
  default: ({ children, href, ...props }: { children: React.ReactNode; href: string }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => {
    const messages: Record<string, string> = {
      "login.title": "Zaloguj się do panelu",
      "login.subtitle": "Wpisz organizację i dane dostępu.",
      "login.organization": "Organizacja",
      "login.organizationPlaceholder": "moja-firma",
      "login.email": "Email",
      "login.password": "Hasło",
      "login.hidePassword": "Ukryj hasło",
      "login.showPassword": "Pokaż hasło",
      "login.submit": "Zaloguj się",
      "login.submitting": "Logowanie...",
      "login.noAccount": "Nie masz konta?",
      "login.register": "Zarejestruj się",
      "login.securityNote": "Dostęp tenant-scoped, gotowy na 2FA i bezpieczną sesję.",
      "twoFa.title": "Weryfikacja 2FA",
      "twoFa.subtitle": "Kod z aplikacji uwierzytelniającej",
      "twoFa.codeLabel": "Kod weryfikacyjny",
      "twoFa.codeHint": "Wpisz 6-cyfrowy kod z aplikacji uwierzytelniającej",
      "twoFa.verify": "Zweryfikuj",
      "twoFa.verifying": "Weryfikacja...",
      "twoFa.backToLogin": "Wróć do logowania",
      "validation.orgRequired": "Slug organizacji jest wymagany",
      "validation.emailInvalid": "Nieprawidłowy adres email",
      "validation.passwordRequired": "Hasło jest wymagane",
    };
    return messages[key] ?? key;
  },
}));

vi.mock("@/hooks/use-auth", () => ({
  useAuth: () => ({
    login: loginMock,
    verify2FALogin: verify2FALoginMock,
  }),
}));

vi.mock("@/hooks/use-public-config", () => ({
  usePublicConfig: () => ({
    registration_mode: "invite",
    license_enabled: true,
  }),
}));

vi.mock("sonner", () => ({
  toast: { error: vi.fn() },
}));

describe("LoginPage Constellation Access", () => {
  beforeEach(() => {
    loginMock.mockReset();
    verify2FALoginMock.mockReset();
    loginMock.mockResolvedValue({ requires2FA: false });
  });

  it("renders the accepted OpenOMS login surface without non-functional extras", () => {
    render(<LoginPage />);

    expect(screen.getAllByText("OpenOMS")[0]).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Zaloguj się do panelu" })).toBeInTheDocument();
    expect(screen.getByText("Wpisz organizację i dane dostępu.")).toBeInTheDocument();
    expect(screen.getByLabelText(/Organizacja/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Email/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Hasło/)).toHaveAttribute("type", "password");
    expect(screen.getByRole("button", { name: "Zaloguj się" })).toBeInTheDocument();
    expect(screen.getByText("Dostęp tenant-scoped, gotowy na 2FA i bezpieczną sesję.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Zarejestruj się" })).toHaveAttribute("href", "/register");
    expect(screen.queryByText(/PL · Polska/)).not.toBeInTheDocument();
  });

  it("keeps the password visibility toggle behavior", async () => {
    const user = userEvent.setup();
    render(<LoginPage />);

    const password = screen.getByLabelText(/Hasło/);
    await user.click(screen.getByRole("button", { name: "Pokaż hasło" }));
    expect(password).toHaveAttribute("type", "text");
    await user.click(screen.getByRole("button", { name: "Ukryj hasło" }));
    expect(password).toHaveAttribute("type", "password");
  });

  it("renders the 2FA card in the same visual shell after login requires a code", async () => {
    const user = userEvent.setup();
    loginMock.mockResolvedValue({ requires2FA: true, tempToken: "temp-token" });

    render(<LoginPage />);
    await user.type(screen.getByLabelText(/Organizacja/), "openoms");
    await user.type(screen.getByLabelText(/Email/), "rafal@openoms.org");
    await user.type(screen.getByLabelText(/Hasło/), "Password123");
    await user.click(screen.getByRole("button", { name: "Zaloguj się" }));

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "Weryfikacja 2FA" })).toBeInTheDocument();
    });
    expect(screen.getByLabelText("Kod weryfikacyjny")).toHaveFocus();
    expect(screen.getByText("Dostęp tenant-scoped, gotowy na 2FA i bezpieczną sesję.")).toBeInTheDocument();
  });
});
```

- [x] **Step 2: Run tests to verify RED**

Run:

```bash
cd apps/dashboard
npx vitest run src/app/'(auth)'/login/page.test.tsx --reporter=dot
```

Expected: fail before implementation because the page does not render the new title, security note, logo image, or Constellation Access surface.

## Task 2: Implement Constellation Login Shell

**Files:**
- Modify: `apps/dashboard/src/app/(auth)/login/page.tsx`
- Copy: `apps/dashboard/public/logos/openoms-community-logo-v2-slack-512.png`
- Modify: `apps/dashboard/messages/pl/layout.json`
- Modify: `apps/dashboard/messages/en/layout.json`

- [x] **Step 1: Copy the brand asset**

Run:

```bash
mkdir -p apps/dashboard/public/logos
cp ../brand-assets/community-logo/openoms-community-logo-v2-slack-512.png apps/dashboard/public/logos/openoms-community-logo-v2-slack-512.png
```

Expected: `apps/dashboard/public/logos/openoms-community-logo-v2-slack-512.png` exists and can be referenced as `/logos/openoms-community-logo-v2-slack-512.png`.

- [x] **Step 2: Update i18n messages**

In `apps/dashboard/messages/pl/layout.json`, set:

```json
"login": {
  "email": "Email",
  "hidePassword": "Ukryj hasło",
  "noAccount": "Nie masz konta?",
  "organization": "Organizacja",
  "organizationPlaceholder": "moja-firma",
  "password": "Hasło",
  "register": "Zarejestruj się",
  "securityNote": "Dostęp tenant-scoped, gotowy na 2FA i bezpieczną sesję.",
  "showPassword": "Pokaż hasło",
  "submit": "Zaloguj się",
  "submitting": "Logowanie...",
  "subtitle": "Wpisz organizację i dane dostępu.",
  "title": "Zaloguj się do panelu"
}
```

In `apps/dashboard/messages/en/layout.json`, set:

```json
"login": {
  "email": "Email",
  "hidePassword": "Hide password",
  "noAccount": "Don't have an account?",
  "organization": "Organization",
  "organizationPlaceholder": "my-company",
  "password": "Password",
  "register": "Register",
  "securityNote": "Tenant-scoped access, ready for 2FA and a secure session.",
  "showPassword": "Show password",
  "submit": "Log in",
  "submitting": "Logging in...",
  "subtitle": "Enter your organization and access details.",
  "title": "Log in to the panel"
}
```

- [x] **Step 3: Replace the old auth presentation**

In `apps/dashboard/src/app/(auth)/login/page.tsx`:
- remove the `Package` import,
- stop using shadcn `Card` wrappers for this route,
- keep `Button`, `Input`, `Label`, `Eye`, `EyeOff`, `ShieldCheck`,
- add route-local components:

```tsx
const logoSrc = "/logos/openoms-community-logo-v2-slack-512.png";

function BrandLockup({ compact = false }: { compact?: boolean }) {
  return (
    <div className={compact ? "flex items-center justify-center gap-3" : "flex items-center gap-3"}>
      <img
        src={logoSrc}
        alt=""
        className={compact ? "h-11 w-11 rounded-[12px]" : "h-9 w-9 rounded-[10px]"}
        width={44}
        height={44}
      />
      <span className={compact ? "text-xl font-semibold text-[#111820]" : "text-[17px] font-semibold text-[#111820]"}>
        OpenOMS
      </span>
    </div>
  );
}

function ConstellationBackground() {
  return (
    <div aria-hidden="true" className="pointer-events-none absolute inset-0 overflow-hidden">
      <div className="absolute left-1/2 top-[47%] h-[520px] w-[520px] -translate-x-1/2 -translate-y-1/2 rounded-full border border-[#05b7c7]/20 auth-orbit auth-orbit-slow" />
      <div className="absolute left-1/2 top-[47%] h-[720px] w-[720px] -translate-x-1/2 -translate-y-1/2 rounded-full border border-[#111820]/10 auth-orbit auth-orbit-medium" />
      <div className="absolute left-1/2 top-[47%] h-[920px] w-[920px] -translate-x-1/2 -translate-y-1/2 rounded-full border border-[#f5a623]/20 auth-orbit auth-orbit-wide" />
      <svg className="absolute inset-0 h-full w-full" viewBox="0 0 1200 820" preserveAspectRatio="none">
        <path className="auth-route" d="M170 510 C360 360, 520 610, 710 410 S980 315, 1080 220" />
        <path className="auth-route auth-route-secondary" d="M210 250 C420 180, 520 300, 650 245 S860 165, 1030 330" />
      </svg>
      <span className="auth-point left-[21%] top-[31%]" />
      <span className="auth-point left-[74%] top-[28%]" />
      <span className="auth-point left-[79%] top-[67%]" />
      <span className="auth-point left-[24%] top-[69%]" />
    </div>
  );
}
```

Add a route shell that owns the full viewport and leaves other auth routes alone:

```tsx
function AuthConstellationShell({ children }: { children: React.ReactNode }) {
  return (
    <main className="fixed inset-0 isolate overflow-y-auto bg-[radial-gradient(circle_at_50%_35%,rgba(32,214,208,0.16),transparent_34%),linear-gradient(135deg,#f8fbfc_0%,#eef4f6_46%,#f6fafb_100%)] px-5 py-8 text-[#18232d] sm:px-6 sm:py-10">
      <ConstellationBackground />
      <div className="absolute left-5 top-5 z-10 sm:left-7 sm:top-7">
        <BrandLockup />
      </div>
      <div className="relative z-10 flex min-h-[calc(100dvh-4rem)] items-center justify-center py-12 sm:min-h-[calc(100dvh-5rem)]">
        {children}
      </div>
      <style jsx global>{`
        .auth-route {
          fill: none;
          stroke: rgba(5, 183, 199, 0.28);
          stroke-width: 1.4;
          stroke-dasharray: 8 16;
          animation: auth-dash 18s linear infinite;
        }
        .auth-route-secondary {
          stroke: rgba(245, 166, 35, 0.2);
          animation-duration: 24s;
        }
        .auth-orbit {
          transform-origin: center;
          animation: auth-spin 34s linear infinite;
        }
        .auth-orbit-medium {
          animation-duration: 44s;
          animation-direction: reverse;
        }
        .auth-orbit-wide {
          animation-duration: 58s;
        }
        .auth-point {
          position: absolute;
          width: 10px;
          height: 10px;
          border-radius: 9999px;
          background: #05b7c7;
          box-shadow: 0 0 0 8px rgba(5, 183, 199, 0.1);
          animation: auth-pulse 3.8s ease-in-out infinite;
        }
        @keyframes auth-dash {
          to { stroke-dashoffset: -160; }
        }
        @keyframes auth-spin {
          to { transform: translate(-50%, -50%) rotate(360deg); }
        }
        @keyframes auth-pulse {
          0%, 100% { opacity: 0.58; transform: scale(1); }
          50% { opacity: 1; transform: scale(1.18); }
        }
        @media (prefers-reduced-motion: reduce) {
          .auth-route,
          .auth-orbit,
          .auth-point {
            animation: none !important;
          }
        }
      `}</style>
    </main>
  );
}
```

Add a reusable card:

```tsx
function AuthCard({ children }: { children: React.ReactNode }) {
  return (
    <section className="w-full max-w-[430px] rounded-[14px] border border-[#d9e2e8]/85 bg-white/90 px-6 py-8 shadow-[0_24px_70px_rgba(17,24,32,0.13)] backdrop-blur-xl sm:px-8 sm:py-9">
      <div className="mb-7 flex flex-col items-center gap-4 text-center">
        <BrandLockup compact />
      </div>
      {children}
    </section>
  );
}
```

- [x] **Step 4: Render the login form in the new shell**

Keep the existing `useForm`, `onSubmit`, `showPassword`, and registration logic. Replace only the returned JSX with:

```tsx
return (
  <AuthConstellationShell>
    <AuthCard>
      <div className="mb-7 text-center">
        <h1 className="text-[27px] font-semibold leading-tight tracking-normal text-[#111820]">
          {t("login.title")}
        </h1>
        <p className="mt-2 text-sm leading-6 text-[#63717d]">
          {t("login.subtitle")}
        </p>
      </div>
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        {/* keep the three existing field groups; restyle labels, inputs, errors, and password toggle */}
        {/* keep the existing conditional register link */}
        <SecurityNote label={t("login.securityNote")} />
      </form>
    </AuthCard>
  </AuthConstellationShell>
);
```

The real implementation must use complete field JSX, not comments. Field styles should use:
- labels: `text-sm font-medium text-[#18232d]`,
- inputs: `h-11 rounded-[8px] border-[#cfdbe2] bg-white text-[#111820] shadow-[0_1px_2px_rgba(17,24,32,0.04)] focus-visible:ring-[#05b7c7]`,
- primary button: `h-11 rounded-[8px] bg-[#111820] text-white hover:bg-[#18232d] focus-visible:ring-[#05b7c7]`,
- errors: `text-xs text-destructive`.

- [x] **Step 5: Run Task 1 tests to verify GREEN**

Run:

```bash
cd apps/dashboard
npx vitest run src/app/'(auth)'/login/page.test.tsx --reporter=dot
```

Expected: all login page tests pass.

## Task 3: Restyle 2FA In The Same Shell

**Files:**
- Modify: `apps/dashboard/src/app/(auth)/login/page.tsx`
- Test: `apps/dashboard/src/app/(auth)/login/page.test.tsx`

- [x] **Step 1: Implement the 2FA return path using `AuthConstellationShell` and `AuthCard`**

When `requires2FA` is true, render:

```tsx
return (
  <AuthConstellationShell>
    <AuthCard>
      <div className="mb-7 text-center">
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full border border-[#05b7c7]/25 bg-[#05b7c7]/10 text-[#007d89]">
          <ShieldCheck className="h-6 w-6" />
        </div>
        <h1 className="text-[27px] font-semibold leading-tight tracking-normal text-[#111820]">
          {t("twoFa.title")}
        </h1>
        <p className="mt-2 text-sm leading-6 text-[#63717d]">
          {t("twoFa.subtitle")}
        </p>
      </div>
      {/* keep existing TOTP input, auto-submit, verify button, and back-to-login behavior */}
      <SecurityNote label={t("login.securityNote")} />
    </AuthCard>
  </AuthConstellationShell>
);
```

The real implementation must keep:
- `ref={codeInputRef}`,
- `inputMode="numeric"`,
- `autoComplete="one-time-code"`,
- `maxLength={6}`,
- `onChange={(e) => handleTotpCodeChange(e.target.value)}`,
- disabled state while verifying,
- back-to-login reset of `requires2FA`, `tempToken`, and `totpCode`.

- [x] **Step 2: Run the 2FA-focused test**

Run:

```bash
cd apps/dashboard
npx vitest run src/app/'(auth)'/login/page.test.tsx --reporter=dot
```

Expected: the 2FA test verifies the heading, focused code input, and shared security note.

## Task 4: Targeted Validation And Visual QA

**Files:**
- Validate all touched files.

- [x] **Step 1: Run targeted lint and tests**

Run:

```bash
cd apps/dashboard
npx eslint --quiet src/app/'(auth)'/login/page.tsx src/app/'(auth)'/login/page.test.tsx
npx vitest run src/app/'(auth)'/login/page.test.tsx --reporter=dot
```

Expected: lint has no output; Vitest passes.

- [x] **Step 2: Run repo diff checks**

Run:

```bash
cd <repository-root>
git diff --check
```

Expected: no whitespace errors.

- [x] **Step 3: Run local dashboard and browser visual QA**

Run:

```bash
cd apps/dashboard
npm run dev
```

Open `/login` in the browser and verify:
- desktop 1440x900: centered 430px card, top-left OpenOMS brand mark, subtle orbit background, no feature tiles, no language switcher,
- mobile 390x844: no horizontal scroll, card readable, fields and buttons not clipped,
- decorative elements are behind content and do not block clicks,
- password toggle works,
- submitting with a mocked/real auth response does not change expected behavior.

- [x] **Step 4: Run full public repo local CI before push**

Run:

```bash
cd <repository-root>
./scripts/local-ci.sh
```

Expected: all checks pass before push.

## Risk And Rollback Notes

- The login page uses `fixed inset-0` to avoid changing the shared auth layout and prevent accidental changes to registration pages. If it causes mobile scroll issues, rollback is limited to `apps/dashboard/src/app/(auth)/login/page.tsx`.
- The decorative CSS is route-local through `style jsx global` selectors prefixed with `auth-`. If it leaks unexpectedly, remove the style block and `ConstellationBackground` without touching auth logic.
- The new asset is static and non-secret. If asset loading fails in production, the login still renders text labels and form controls; rollback by removing the `<img>` and using text-only `BrandLockup`.
- No backend, API, database, or production configuration changes are part of this task.

## Self-Review

- Spec coverage: central card, OpenOMS logo, constellation/orbits, no language switcher, no feature tiles, preserved login and 2FA behavior, mobile/readability, reduced motion, and visual QA are all mapped to tasks.
- Placeholder scan: no task depends on unspecified future work; implementation comments in snippets are explicitly expanded in surrounding instructions.
- Type consistency: `loginMock`, `verify2FALoginMock`, `AuthConstellationShell`, `AuthCard`, `BrandLockup`, `ConstellationBackground`, and `SecurityNote` names are used consistently.
