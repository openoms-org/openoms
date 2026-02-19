# OpenOMS Dashboard — Design System

## Direction

Polish e-commerce OMS dashboard. The user is an operations manager or warehouse worker handling orders, shipments, and inventory in a fast-paced fulfillment environment. The interface should feel **dense but calm** — like a well-organized control room. Functional, not decorative.

## Depth Strategy

**Borders-only.** No shadows anywhere. Cards, dialogs, dropdowns — all use `border` for separation. Higher elevation surfaces use slightly shifted background lightness, never box-shadow.

- Cards: `rounded-xl border`, no shadow
- Dialogs: `rounded-lg border`, no shadow
- Dropdowns: border + slight background shift
- Inputs: slightly darker than surroundings (inset feel)

## Color

OKLch-based tokens, hue 250 (blue) throughout. Sidebar shares the same hue as the content area with subtle lightness shift.

- Light sidebar: `oklch(0.975 0.005 250)`
- Dark sidebar: `oklch(0.16 0.015 250)`
- Accent/primary: `oklch(0.35 0.08 250)` light / `oklch(0.65 0.12 250)` dark
- Single accent color. No multiple hues for decoration.

## Typography

- Table headers: `text-xs font-medium text-muted-foreground` — sentence case, no uppercase, no tracking-wider
- Group headers (sidebar): `text-xs font-medium text-muted-foreground` — sentence case
- Nav items: `text-sm font-medium`
- Card titles: `font-semibold leading-none`

## Spacing

Base unit: 4px (Tailwind default). Card internal padding: `p-4`. Card gap: `gap-4`. Empty states: `py-10`. Sidebar nav items: `px-3 py-2`. Sidebar groups: `mt-3 first:mt-0`.

## Cards

`gap-4 py-4` on Card. `px-4` on CardHeader, CardContent, CardFooter. `[.border-b]:pb-4` and `[.border-t]:pt-4` for divided variants.

## Tables

Headers: sentence case, `text-xs font-medium`. No uppercase, no letter-spacing. Sticky `top-0 z-10` with background match. Rows: `hover:bg-muted/50` transition.

## Sidebar

- Same background hue as content, separated by `border-r`
- Active items: `border-l-2 border-sidebar-primary bg-sidebar-accent`
- Inactive items: `border-l-2 border-transparent` (prevents layout shift)
- 8 collapsible groups, single-item groups merged into neighbors
- Group toggle: ChevronRight icon with rotate-90 transition
- Collapsed mode: single TooltipProvider, icon-only with tooltip on hover
- Keyboard shortcut: Ctrl+B to toggle

## Empty States

`py-10`, centered. 16x16 muted circle with 8x8 icon. Title + description + optional action buttons.

## Navigation

- 8 groups: Sprzedaz, Katalog, Logistyka, Kanaly sprzedazy, Raporty, Zaopatrzenie, Narzedzia, Ustawienia
- No single-item groups — merge into neighbors
- Every nav icon unique — no duplicates across items
- Active state detection: shared `isNavItemActive()` utility with prefix matching and sibling specificity check
