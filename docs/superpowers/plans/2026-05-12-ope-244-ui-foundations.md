# OPE-244 UI Foundations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first reusable UI foundation slice for the OPE-260 dashboard-wide visual system.

**Architecture:** Keep the existing OPE-242 shell and shadcn/ui primitives. Add focused shared components in `apps/dashboard/src/components/shared/` that compose existing `ui/*` primitives and preserve backward compatibility with current screens. This PR should not migrate every route; it creates tested foundations that subsequent PRs can use safely.

**Tech Stack:** Next.js 16 App Router, React 19, TypeScript, Tailwind v4, shadcn/ui, lucide-react, Vitest, Testing Library.

---

## OPE-260 Program Roadmap

This plan covers the first implementation PR only: **OPE-244 UI foundations**.

Follow-up implementation plans should be created after this PR lands:

1. `OPE-260/OPE-244`: UI foundations, shared primitives, compatibility pass.
2. Operator lists: orders, shipments, products, returns, customers.
3. Provider and logistics surfaces: integrations, marketplaces, carriers, suppliers, stock sync.
4. Admin and settings surfaces: settings layout, users, roles, security, billing, webhooks, sync.
5. Auth, onboarding, and public flows.
6. Final polish: accessibility, responsive QA, command palette/nav readiness leaks.

## File Structure

Create:

- `apps/dashboard/src/components/shared/surface.tsx`
  Shared white/muted/status surface wrapper used by sections, detail panels, and settings panels.

- `apps/dashboard/src/components/shared/page-section.tsx`
  Section wrapper with title, description, optional action area, and optional surface framing.

- `apps/dashboard/src/components/shared/action-bar.tsx`
  Reusable action row for page headers, forms, and table toolbars.

- `apps/dashboard/src/components/shared/form-layout.tsx`
  Form section and action layout primitives. It does not own form state.

- `apps/dashboard/src/components/shared/detail-layout.tsx`
  Reusable two-column detail layout for order, shipment, product, return, integration, and customer detail pages.

- `apps/dashboard/src/components/shared/settings-layout.tsx`
  Reusable settings page layout with local navigation/sections.

- `apps/dashboard/src/components/shared/__tests__/layout-primitives.test.tsx`
  Unit tests for the new layout primitives.

Modify:

- `apps/dashboard/src/components/shared/page-header.tsx`
  Preserve existing `action` prop while adding `actions`, `breadcrumbs`, `meta`, and compact visual style.

- `apps/dashboard/src/components/shared/empty-state.tsx`
  Preserve existing props while adding safer action rendering, compact variant, and accessible icon handling.

- `apps/dashboard/src/components/shared/status-badge.tsx`
  Preserve current `statusMap` behavior while adding semantic tone support for future screens.

- `apps/dashboard/src/components/shared/data-table.tsx`
  Improve accessibility and visual consistency without changing call sites.

- `apps/dashboard/src/components/shared/__tests__/data-table.test.tsx`
  Add tests for checkbox labels and sortable button labels.

- `apps/dashboard/src/components/shared/__tests__/status-badge.test.tsx`
  Add tests for semantic tones and legacy status map compatibility.

Do not modify route pages in this PR unless a shared component change forces a small compatibility fix.

## Task 1: Add Surface, PageSection, and ActionBar

**Files:**
- Create: `apps/dashboard/src/components/shared/surface.tsx`
- Create: `apps/dashboard/src/components/shared/page-section.tsx`
- Create: `apps/dashboard/src/components/shared/action-bar.tsx`
- Create: `apps/dashboard/src/components/shared/__tests__/layout-primitives.test.tsx`

- [ ] **Step 1: Write failing tests for the new structural primitives**

Add this initial test file:

```tsx
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { Surface } from "@/components/shared/surface";
import { PageSection } from "@/components/shared/page-section";
import { ActionBar } from "@/components/shared/action-bar";
import { Button } from "@/components/ui/button";

describe("layout primitives", () => {
  it("renders Surface as a named region when aria-label is provided", () => {
    render(
      <Surface aria-label="Order summary">
        <p>Summary content</p>
      </Surface>
    );

    expect(screen.getByRole("region", { name: "Order summary" })).toBeInTheDocument();
    expect(screen.getByText("Summary content")).toBeInTheDocument();
  });

  it("renders PageSection with title, description, actions, and children", () => {
    render(
      <PageSection
        title="Shipments"
        description="Labels and tracking"
        actions={<Button>New shipment</Button>}
      >
        <p>Shipment table</p>
      </PageSection>
    );

    expect(screen.getByRole("heading", { name: "Shipments" })).toBeInTheDocument();
    expect(screen.getByText("Labels and tracking")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New shipment" })).toBeInTheDocument();
    expect(screen.getByText("Shipment table")).toBeInTheDocument();
  });

  it("renders ActionBar with primary and secondary action areas", () => {
    render(
      <ActionBar
        secondary={<Button variant="outline">Cancel</Button>}
        primary={<Button>Save</Button>}
      />
    );

    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the tests and verify they fail**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/layout-primitives.test.tsx --reporter=dot
```

Expected: fail because `surface`, `page-section`, and `action-bar` modules do not exist.

- [ ] **Step 3: Create `Surface`**

Create `apps/dashboard/src/components/shared/surface.tsx`:

```tsx
import * as React from "react";
import { cn } from "@/lib/utils";

type SurfaceTone = "default" | "muted" | "success" | "warning" | "danger";

const toneClasses: Record<SurfaceTone, string> = {
  default: "border-border bg-card text-card-foreground",
  muted: "border-border bg-muted/35 text-foreground",
  success: "border-emerald-200 bg-emerald-50 text-emerald-950",
  warning: "border-amber-200 bg-amber-50 text-amber-950",
  danger: "border-red-200 bg-red-50 text-red-950",
};

interface SurfaceProps extends React.ComponentProps<"section"> {
  tone?: SurfaceTone;
  padded?: boolean;
}

export function Surface({
  tone = "default",
  padded = true,
  className,
  "aria-label": ariaLabel,
  "aria-labelledby": ariaLabelledBy,
  ...props
}: SurfaceProps) {
  return (
    <section
      role={ariaLabel || ariaLabelledBy ? "region" : undefined}
      aria-label={ariaLabel}
      aria-labelledby={ariaLabelledBy}
      className={cn(
        "rounded-lg border",
        toneClasses[tone],
        padded && "p-4 sm:p-5",
        className
      )}
      {...props}
    />
  );
}
```

- [ ] **Step 4: Create `PageSection`**

Create `apps/dashboard/src/components/shared/page-section.tsx`:

```tsx
import * as React from "react";
import { Surface } from "@/components/shared/surface";
import { cn } from "@/lib/utils";

interface PageSectionProps extends React.ComponentProps<"section"> {
  title?: string;
  description?: string;
  actions?: React.ReactNode;
  framed?: boolean;
  contentClassName?: string;
}

export function PageSection({
  title,
  description,
  actions,
  framed = true,
  className,
  contentClassName,
  children,
  ...props
}: PageSectionProps) {
  const hasHeader = title || description || actions;
  const content = (
    <>
      {hasHeader && (
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            {title && (
              <h2 className="text-base font-semibold tracking-normal text-foreground">
                {title}
              </h2>
            )}
            {description && (
              <p className="mt-1 text-sm leading-6 text-muted-foreground">
                {description}
              </p>
            )}
          </div>
          {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
        </div>
      )}
      <div className={contentClassName}>{children}</div>
    </>
  );

  if (framed) {
    return (
      <Surface className={className} {...props}>
        {content}
      </Surface>
    );
  }

  return (
    <section className={cn("space-y-4", className)} {...props}>
      {content}
    </section>
  );
}
```

- [ ] **Step 5: Create `ActionBar`**

Create `apps/dashboard/src/components/shared/action-bar.tsx`:

```tsx
import * as React from "react";
import { cn } from "@/lib/utils";

interface ActionBarProps extends React.ComponentProps<"div"> {
  primary?: React.ReactNode;
  secondary?: React.ReactNode;
  meta?: React.ReactNode;
  sticky?: boolean;
}

export function ActionBar({
  primary,
  secondary,
  meta,
  sticky = false,
  className,
  ...props
}: ActionBarProps) {
  return (
    <div
      className={cn(
        "flex flex-col gap-3 border-border bg-background/95 py-3 sm:flex-row sm:items-center sm:justify-between",
        sticky && "sticky bottom-0 z-10 border-t backdrop-blur",
        className
      )}
      {...props}
    >
      <div className="flex min-w-0 flex-1 items-center gap-2 text-sm text-muted-foreground">
        {meta}
      </div>
      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-end">
        {secondary}
        {primary}
      </div>
    </div>
  );
}
```

- [ ] **Step 6: Run tests and verify they pass**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/layout-primitives.test.tsx --reporter=dot
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add apps/dashboard/src/components/shared/surface.tsx \
  apps/dashboard/src/components/shared/page-section.tsx \
  apps/dashboard/src/components/shared/action-bar.tsx \
  apps/dashboard/src/components/shared/__tests__/layout-primitives.test.tsx
git commit -m "OPE-244: add dashboard layout primitives"
```

## Task 2: Upgrade PageHeader and EmptyState

**Files:**
- Modify: `apps/dashboard/src/components/shared/page-header.tsx`
- Modify: `apps/dashboard/src/components/shared/empty-state.tsx`
- Modify: `apps/dashboard/src/components/shared/__tests__/layout-primitives.test.tsx`

- [ ] **Step 1: Add failing tests for backward-compatible header and empty state behavior**

Append these tests to `layout-primitives.test.tsx`:

```tsx
import { PackagePlus } from "lucide-react";
import { EmptyState } from "@/components/shared/empty-state";
import { PageHeader } from "@/components/shared/page-header";

it("renders PageHeader legacy action and additional action slots", () => {
  render(
    <PageHeader
      title="Products"
      description="Catalog orchestration"
      action={{ label: "Add product", href: "/products/new" }}
      actions={<Button variant="outline">Import</Button>}
      meta={<span>12 active</span>}
    />
  );

  expect(screen.getByRole("heading", { name: "Products" })).toBeInTheDocument();
  expect(screen.getByText("Catalog orchestration")).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Add product" })).toHaveAttribute("href", "/products/new");
  expect(screen.getByRole("button", { name: "Import" })).toBeInTheDocument();
  expect(screen.getByText("12 active")).toBeInTheDocument();
});

it("renders EmptyState compact variant with accessible icon wrapper", () => {
  render(
    <EmptyState
      icon={PackagePlus}
      title="No products"
      description="Create the first catalog item."
      action={{ label: "Add product", href: "/products/new" }}
      variant="compact"
    />
  );

  expect(screen.getByRole("heading", { name: "No products" })).toBeInTheDocument();
  expect(screen.getByText("Create the first catalog item.")).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Add product" })).toHaveAttribute("href", "/products/new");
  expect(screen.getByTestId("empty-state-icon")).toHaveAttribute("aria-hidden", "true");
});
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/layout-primitives.test.tsx --reporter=dot
```

Expected: fail because `PageHeader` does not accept `actions`/`meta`, and `EmptyState` does not accept `variant`.

- [ ] **Step 3: Replace `PageHeader` with a compatible system version**

Replace `apps/dashboard/src/components/shared/page-header.tsx` with:

```tsx
import * as React from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface PageHeaderAction {
  label: string;
  href: string;
}

interface PageHeaderProps {
  title: string;
  description?: string;
  action?: PageHeaderAction;
  actions?: React.ReactNode;
  breadcrumbs?: React.ReactNode;
  meta?: React.ReactNode;
  className?: string;
}

export function PageHeader({
  title,
  description,
  action,
  actions,
  breadcrumbs,
  meta,
  className,
}: PageHeaderProps) {
  return (
    <header className={cn("mb-6 space-y-3", className)}>
      {breadcrumbs}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 space-y-2">
          <div className="space-y-1">
            <h1 className="text-2xl font-semibold tracking-normal text-foreground">
              {title}
            </h1>
            {description && (
              <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
                {description}
              </p>
            )}
          </div>
          {meta && <div className="text-sm text-muted-foreground">{meta}</div>}
        </div>
        {(action || actions) && (
          <div className="flex shrink-0 flex-col-reverse gap-2 sm:flex-row sm:items-center">
            {actions}
            {action && (
              <Button asChild>
                <Link href={action.href}>{action.label}</Link>
              </Button>
            )}
          </div>
        )}
      </div>
    </header>
  );
}
```

- [ ] **Step 4: Replace `EmptyState` with a compatible system version**

Replace `apps/dashboard/src/components/shared/empty-state.tsx` with:

```tsx
import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface EmptyStateAction {
  label: string;
  href: string;
}

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: EmptyStateAction;
  secondaryAction?: EmptyStateAction;
  variant?: "default" | "compact";
  className?: string;
}

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  secondaryAction,
  variant = "default",
  className,
}: EmptyStateProps) {
  const isCompact = variant === "compact";

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center",
        isCompact ? "py-8" : "py-12",
        className
      )}
    >
      <div
        data-testid="empty-state-icon"
        aria-hidden="true"
        className={cn(
          "mb-4 flex items-center justify-center rounded-full border bg-muted/50 text-muted-foreground",
          isCompact ? "h-12 w-12" : "h-16 w-16"
        )}
      >
        <Icon className={isCompact ? "h-5 w-5" : "h-7 w-7"} />
      </div>
      <h3 className={cn("font-semibold text-foreground", isCompact ? "text-base" : "text-lg")}>
        {title}
      </h3>
      <p className="mt-1 max-w-sm text-sm leading-6 text-muted-foreground">
        {description}
      </p>
      {(action || secondaryAction) && (
        <div className="mt-5 flex flex-col gap-2 sm:flex-row">
          {action && (
            <Button asChild>
              <Link href={action.href}>{action.label}</Link>
            </Button>
          )}
          {secondaryAction && (
            <Button asChild variant="outline">
              <Link href={secondaryAction.href}>{secondaryAction.label}</Link>
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 5: Run tests and verify they pass**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/layout-primitives.test.tsx --reporter=dot
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add apps/dashboard/src/components/shared/page-header.tsx \
  apps/dashboard/src/components/shared/empty-state.tsx \
  apps/dashboard/src/components/shared/__tests__/layout-primitives.test.tsx
git commit -m "OPE-244: standardize page headers and empty states"
```

## Task 3: Add FormLayout, DetailLayout, and SettingsLayout

**Files:**
- Create: `apps/dashboard/src/components/shared/form-layout.tsx`
- Create: `apps/dashboard/src/components/shared/detail-layout.tsx`
- Create: `apps/dashboard/src/components/shared/settings-layout.tsx`
- Modify: `apps/dashboard/src/components/shared/__tests__/layout-primitives.test.tsx`

- [ ] **Step 1: Add failing tests for form, detail, and settings layouts**

Append:

```tsx
import {
  FormActions,
  FormSection,
} from "@/components/shared/form-layout";
import {
  DetailLayout,
  DetailMain,
  DetailSidebar,
} from "@/components/shared/detail-layout";
import {
  SettingsLayout,
  SettingsNav,
  SettingsPanel,
} from "@/components/shared/settings-layout";

it("renders FormSection and FormActions", () => {
  render(
    <FormSection title="Carrier credentials" description="Production API settings">
      <label htmlFor="token">Token</label>
      <input id="token" />
      <FormActions primary={<Button>Save</Button>} secondary={<Button variant="outline">Cancel</Button>} />
    </FormSection>
  );

  expect(screen.getByRole("heading", { name: "Carrier credentials" })).toBeInTheDocument();
  expect(screen.getByText("Production API settings")).toBeInTheDocument();
  expect(screen.getByLabelText("Token")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
});

it("renders DetailLayout with main and sidebar regions", () => {
  render(
    <DetailLayout>
      <DetailMain aria-label="Order activity">Activity</DetailMain>
      <DetailSidebar aria-label="Order metadata">Metadata</DetailSidebar>
    </DetailLayout>
  );

  expect(screen.getByRole("region", { name: "Order activity" })).toHaveTextContent("Activity");
  expect(screen.getByRole("region", { name: "Order metadata" })).toHaveTextContent("Metadata");
});

it("renders SettingsLayout with navigation and panel", () => {
  render(
    <SettingsLayout>
      <SettingsNav aria-label="Settings sections">
        <a href="/settings/company">Company</a>
      </SettingsNav>
      <SettingsPanel title="Company" description="Company profile">
        Settings content
      </SettingsPanel>
    </SettingsLayout>
  );

  expect(screen.getByRole("navigation", { name: "Settings sections" })).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "Company" })).toBeInTheDocument();
  expect(screen.getByText("Settings content")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/layout-primitives.test.tsx --reporter=dot
```

Expected: fail because layout modules do not exist.

- [ ] **Step 3: Create `form-layout.tsx`**

```tsx
import * as React from "react";
import { ActionBar } from "@/components/shared/action-bar";
import { Surface } from "@/components/shared/surface";
import { cn } from "@/lib/utils";

interface FormSectionProps extends React.ComponentProps<"section"> {
  title: string;
  description?: string;
}

export function FormSection({
  title,
  description,
  className,
  children,
  ...props
}: FormSectionProps) {
  return (
    <Surface className={cn("space-y-5", className)} {...props}>
      <div>
        <h2 className="text-base font-semibold text-foreground">{title}</h2>
        {description && (
          <p className="mt-1 text-sm leading-6 text-muted-foreground">{description}</p>
        )}
      </div>
      <div className="grid gap-4">{children}</div>
    </Surface>
  );
}

interface FormActionsProps {
  primary?: React.ReactNode;
  secondary?: React.ReactNode;
  meta?: React.ReactNode;
  sticky?: boolean;
}

export function FormActions({ primary, secondary, meta, sticky }: FormActionsProps) {
  return (
    <ActionBar
      className="mt-2"
      primary={primary}
      secondary={secondary}
      meta={meta}
      sticky={sticky}
    />
  );
}
```

- [ ] **Step 4: Create `detail-layout.tsx`**

```tsx
import * as React from "react";
import { cn } from "@/lib/utils";

export function DetailLayout({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]", className)}
      {...props}
    />
  );
}

export function DetailMain({
  className,
  "aria-label": ariaLabel,
  "aria-labelledby": ariaLabelledBy,
  ...props
}: React.ComponentProps<"section">) {
  return (
    <section
      role={ariaLabel || ariaLabelledBy ? "region" : undefined}
      aria-label={ariaLabel}
      aria-labelledby={ariaLabelledBy}
      className={cn("min-w-0 space-y-4", className)}
      {...props}
    />
  );
}

export function DetailSidebar({
  className,
  "aria-label": ariaLabel,
  "aria-labelledby": ariaLabelledBy,
  ...props
}: React.ComponentProps<"aside">) {
  return (
    <aside
      role={ariaLabel || ariaLabelledBy ? "region" : undefined}
      aria-label={ariaLabel}
      aria-labelledby={ariaLabelledBy}
      className={cn("min-w-0 space-y-4", className)}
      {...props}
    />
  );
}
```

- [ ] **Step 5: Create `settings-layout.tsx`**

```tsx
import * as React from "react";
import { Surface } from "@/components/shared/surface";
import { cn } from "@/lib/utils";

export function SettingsLayout({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("grid gap-6 lg:grid-cols-[260px_minmax(0,1fr)]", className)}
      {...props}
    />
  );
}

export function SettingsNav({
  className,
  ...props
}: React.ComponentProps<"nav">) {
  return (
    <nav
      className={cn("space-y-1 lg:sticky lg:top-20 lg:self-start", className)}
      {...props}
    />
  );
}

interface SettingsPanelProps extends React.ComponentProps<"section"> {
  title: string;
  description?: string;
  actions?: React.ReactNode;
}

export function SettingsPanel({
  title,
  description,
  actions,
  className,
  children,
  ...props
}: SettingsPanelProps) {
  return (
    <Surface className={cn("space-y-5", className)} {...props}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-base font-semibold text-foreground">{title}</h2>
          {description && (
            <p className="mt-1 text-sm leading-6 text-muted-foreground">{description}</p>
          )}
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
      <div>{children}</div>
    </Surface>
  );
}
```

- [ ] **Step 6: Run tests and verify they pass**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/layout-primitives.test.tsx --reporter=dot
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add apps/dashboard/src/components/shared/form-layout.tsx \
  apps/dashboard/src/components/shared/detail-layout.tsx \
  apps/dashboard/src/components/shared/settings-layout.tsx \
  apps/dashboard/src/components/shared/__tests__/layout-primitives.test.tsx
git commit -m "OPE-244: add form detail and settings layouts"
```

## Task 4: Add semantic StatusBadge tones

**Files:**
- Modify: `apps/dashboard/src/components/shared/status-badge.tsx`
- Modify: `apps/dashboard/src/components/shared/__tests__/status-badge.test.tsx`

- [ ] **Step 1: Add failing tests for semantic tones**

Append:

```tsx
it("renders semantic warning tone without a legacy status map", () => {
  render(<StatusBadge status="warning" label="Needs review" tone="warning" />);
  const badge = screen.getByText("Needs review");
  expect(badge).toHaveClass("bg-amber-50");
  expect(badge).toHaveClass("text-amber-800");
});

it("keeps legacy statusMap classes when tone is not provided", () => {
  const { container } = render(<StatusBadge status="new" statusMap={ORDER_STATUSES} />);
  const badge = container.querySelector("span");
  expect(badge).toHaveClass("bg-blue-100");
  expect(badge).toHaveClass("text-blue-800");
});
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/status-badge.test.tsx --reporter=dot
```

Expected: fail because `label` and `tone` props do not exist.

- [ ] **Step 3: Update `StatusBadge`**

Replace `apps/dashboard/src/components/shared/status-badge.tsx` with:

```tsx
"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type StatusTone = "ok" | "warning" | "problem" | "inactive" | "draft" | "pending" | "blocked";

const toneClasses: Record<StatusTone, string> = {
  ok: "bg-emerald-50 text-emerald-800 ring-1 ring-emerald-200",
  warning: "bg-amber-50 text-amber-800 ring-1 ring-amber-200",
  problem: "bg-red-50 text-red-800 ring-1 ring-red-200",
  inactive: "bg-slate-100 text-slate-700 ring-1 ring-slate-200",
  draft: "bg-slate-100 text-slate-700 ring-1 ring-slate-200",
  pending: "bg-blue-50 text-blue-800 ring-1 ring-blue-200",
  blocked: "bg-red-50 text-red-800 ring-1 ring-red-200",
};

interface StatusBadgeProps {
  status: string;
  statusMap?: Record<string, { label: string; color: string }>;
  /** Translation prefix under "statuses" namespace, e.g. "order", "shipment", "return" */
  translationPrefix?: string;
  label?: string;
  tone?: StatusTone;
}

export function StatusBadge({
  status,
  statusMap,
  translationPrefix,
  label: explicitLabel,
  tone,
}: StatusBadgeProps) {
  const t = useTranslations("statuses");
  const config = statusMap?.[status];

  let label = explicitLabel ?? config?.label ?? status;
  if (!explicitLabel && translationPrefix) {
    try {
      label = t(`${translationPrefix}.${status}`);
    } catch {
      label = config?.label ?? status;
    }
  }

  if (tone) {
    return (
      <span
        className={cn(
          "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
          toneClasses[tone]
        )}
      >
        {label}
      </span>
    );
  }

  if (config) {
    return (
      <span
        className={cn(
          "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
          config.color
        )}
      >
        {label}
      </span>
    );
  }

  return <Badge variant="outline">{label}</Badge>;
}
```

- [ ] **Step 4: Run tests and verify they pass**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/status-badge.test.tsx --reporter=dot
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add apps/dashboard/src/components/shared/status-badge.tsx \
  apps/dashboard/src/components/shared/__tests__/status-badge.test.tsx
git commit -m "OPE-244: add semantic status badge tones"
```

## Task 5: Improve DataTable accessibility without changing call sites

**Files:**
- Modify: `apps/dashboard/src/components/shared/data-table.tsx`
- Modify: `apps/dashboard/src/components/shared/__tests__/data-table.test.tsx`

- [ ] **Step 1: Add failing accessibility tests**

Append:

```tsx
it("labels the select-all checkbox and row checkboxes", () => {
  render(
    <DataTable
      columns={columns}
      data={testData}
      selectable
      selectedIds={new Set()}
      onSelectionChange={() => {}}
    />
  );

  expect(screen.getByRole("checkbox", { name: "Select all rows" })).toBeInTheDocument();
  expect(screen.getByRole("checkbox", { name: "Select row Jan Kowalski" })).toBeInTheDocument();
  expect(screen.getByRole("checkbox", { name: "Select row Anna Nowak" })).toBeInTheDocument();
});

it("labels sortable header buttons", () => {
  const sortableColumns: ColumnDef<TestRow>[] = [
    { header: "Name", accessorKey: "name", sortable: true },
    { header: "Email", accessorKey: "email" },
  ];

  render(
    <DataTable
      columns={sortableColumns}
      data={testData}
      sortBy="name"
      sortOrder="asc"
      onSort={() => {}}
    />
  );

  expect(screen.getByRole("button", { name: "Sort by Name" })).toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/data-table.test.tsx --reporter=dot
```

Expected: fail because checkboxes and sortable buttons do not have accessible labels.

- [ ] **Step 3: Update sortable button labels**

In both loading and non-loading table header branches, change sortable buttons from:

```tsx
<button
  className="flex items-center gap-1 hover:text-foreground"
  onClick={() => onSort(String(column.accessorKey))}
>
```

to:

```tsx
<button
  type="button"
  aria-label={`Sort by ${column.header}`}
  className="flex items-center gap-1 hover:text-foreground"
  onClick={() => onSort(String(column.accessorKey))}
>
```

For the non-loading branch where `key` is already available, use:

```tsx
<button
  type="button"
  aria-label={`Sort by ${column.header}`}
  className="flex items-center gap-1 hover:text-foreground"
  onClick={() => onSort(key)}
>
```

- [ ] **Step 4: Update selectable checkbox labels**

Change the select-all checkbox to:

```tsx
<input
  type="checkbox"
  aria-label="Select all rows"
  className="cursor-pointer"
  checked={allSelected}
  ref={(el) => {
    if (el) el.indeterminate = someSelected && !allSelected;
  }}
  onChange={toggleAll}
/>
```

Change the row checkbox to:

```tsx
<input
  type="checkbox"
  aria-label={`Select row ${String(getNestedValue(row, "name") ?? id)}`}
  className="cursor-pointer"
  checked={selectedIds?.has(id) || false}
  onChange={() => toggleRow(id)}
  onClick={(e) => e.stopPropagation()}
/>
```

This uses `name` when present and falls back to row ID. It is backward-compatible with existing tables.

- [ ] **Step 5: Run tests and verify they pass**

Run:

```bash
cd apps/dashboard
npx vitest run src/components/shared/__tests__/data-table.test.tsx --reporter=dot
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add apps/dashboard/src/components/shared/data-table.tsx \
  apps/dashboard/src/components/shared/__tests__/data-table.test.tsx
git commit -m "OPE-244: improve data table accessibility"
```

## Task 6: Verification and documentation

**Files:**
- Modify: `docs/superpowers/specs/2026-05-12-ope-260-dashboard-ui-system.md`

- [ ] **Step 1: Add implementation status to the OPE-260 spec**

Append this section to `docs/superpowers/specs/2026-05-12-ope-260-dashboard-ui-system.md`:

```markdown
## Status

2026-05-12: OPE-244 began the first implementation slice for the UI system. The slice adds shared layout primitives and backward-compatible upgrades to PageHeader, EmptyState, StatusBadge and DataTable, so future route migrations can use one visual system instead of per-page styling.
```

- [ ] **Step 2: Run focused tests**

Run:

```bash
cd apps/dashboard
npx vitest run \
  src/components/shared/__tests__/layout-primitives.test.tsx \
  src/components/shared/__tests__/status-badge.test.tsx \
  src/components/shared/__tests__/data-table.test.tsx \
  --reporter=dot
```

Expected: all tests pass.

- [ ] **Step 3: Run frontend lint for touched files**

Run:

```bash
cd apps/dashboard
npx eslint --quiet \
  src/components/shared/surface.tsx \
  src/components/shared/page-section.tsx \
  src/components/shared/action-bar.tsx \
  src/components/shared/form-layout.tsx \
  src/components/shared/detail-layout.tsx \
  src/components/shared/settings-layout.tsx \
  src/components/shared/page-header.tsx \
  src/components/shared/empty-state.tsx \
  src/components/shared/status-badge.tsx \
  src/components/shared/data-table.tsx \
  src/components/shared/__tests__/layout-primitives.test.tsx \
  src/components/shared/__tests__/status-badge.test.tsx \
  src/components/shared/__tests__/data-table.test.tsx
```

Expected: no errors.

- [ ] **Step 4: Run Next build**

Run:

```bash
cd apps/dashboard
NEXT_TELEMETRY_DISABLED=1 npx next build
```

Expected: build completes successfully.

- [ ] **Step 5: Run diff check**

Run:

```bash
git diff --check
```

Expected: no whitespace errors.

- [ ] **Step 6: Commit docs update**

```bash
git add docs/superpowers/specs/2026-05-12-ope-260-dashboard-ui-system.md
git commit -m "OPE-244: update UI system implementation status"
```

- [ ] **Step 7: Run full local CI before push**

Run:

```bash
./scripts/local-ci.sh
```

Expected: all checks pass. Save the result summary in the terminal notes for the PR.

## PR Preparation

- [ ] Push branch:

```bash
git push -u origin feat/OPE-244-ui-foundations
```

- [ ] Open PR with title:

```text
OPE-244: establish dashboard UI foundations
```

- [ ] PR body must include:

```markdown
## Summary
- Adds shared dashboard UI primitives for the OPE-260 visual system.
- Keeps existing PageHeader, EmptyState, StatusBadge and DataTable call sites backward-compatible.
- Improves DataTable accessible labels for selection and sorting.

## Test plan
- `cd apps/dashboard && npx vitest run src/components/shared/__tests__/layout-primitives.test.tsx src/components/shared/__tests__/status-badge.test.tsx src/components/shared/__tests__/data-table.test.tsx --reporter=dot`
- `cd apps/dashboard && npx eslint --quiet src/components/shared/surface.tsx src/components/shared/page-section.tsx src/components/shared/action-bar.tsx src/components/shared/form-layout.tsx src/components/shared/detail-layout.tsx src/components/shared/settings-layout.tsx src/components/shared/page-header.tsx src/components/shared/empty-state.tsx src/components/shared/status-badge.tsx src/components/shared/data-table.tsx src/components/shared/__tests__/layout-primitives.test.tsx src/components/shared/__tests__/status-badge.test.tsx src/components/shared/__tests__/data-table.test.tsx`
- `cd apps/dashboard && NEXT_TELEMETRY_DISABLED=1 npx next build`
- `git diff --check`
- `./scripts/local-ci.sh`

## Docs updated
- [x] `docs/superpowers/specs/2026-05-12-ope-260-dashboard-ui-system.md` — added OPE-244 implementation status
```

## Self-Review Checklist

- [ ] The plan covers the OPE-260 requirement to start with shared primitives before route migration.
- [ ] The plan preserves current route behavior by keeping existing shared component props backward-compatible.
- [ ] The plan does not expose hidden or unready features.
- [ ] The plan has focused tests for every new primitive and changed shared behavior.
- [ ] The plan leaves hardening work such as OPE-259 for a separate PR.
