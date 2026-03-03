# Frontend Components Audit

**Total: ~75 business components across 16 subdirectories**

---

## DASHBOARD ANALYTICS (`/components/dashboard/`)

| Component | File | Purpose |
|-----------|------|---------|
| StatCards | `stat-cards.tsx` | KPI metric cards (total, new, in-transit, delivered) |
| OrderStatusChart | `order-status-chart.tsx` | Recharts BarChart — order distribution by status |
| OrderSourceChart | `order-source-chart.tsx` | PieChart — orders by marketplace source |
| RevenueChart | `revenue-chart.tsx` | AreaChart — 30-day revenue trend |
| RecentOrdersTable | `recent-orders-table.tsx` | Latest orders quick table |

---

## ORDERS (`/components/orders/`)

| Component | File | Purpose |
|-----------|------|---------|
| KanbanBoard | `kanban-board.tsx` | Drag-and-drop order workflow (dnd-kit), infinite scroll per column |
| KanbanCard | `kanban-card.tsx` | Individual draggable order card |
| OrderForm | `order-form.tsx` | Order creation/editing — dynamic custom fields, line items, address, InPost |
| OrderFilters | `order-filters.tsx` | Status/source/payment/priority filter controls |
| OrderStatusActions | `order-status-actions.tsx` | Status transition buttons (normal + force) |
| OrderTimeline | `order-timeline.tsx` | Audit log timeline for order history |
| BulkActions | `bulk-actions.tsx` | Bulk status transition for selected orders |

---

## SHIPMENTS (`/components/shipments/`)

| Component | File | Purpose |
|-----------|------|---------|
| ShipmentForm | `shipment-form.tsx` | Create/edit shipment with carrier selection |
| ShipmentFilters | `shipment-filters.tsx` | Status/provider filter controls |
| ShipmentStatusActions | `shipment-status-actions.tsx` | Shipment status transition buttons |
| CarrierFields | `carrier-fields.tsx` | Polymorphic carrier-specific form (InPost, DHL, GLS, UPS, etc.) |
| DispatchOrderDialog | `dispatch-order-dialog.tsx` | InPost courier pickup request |
| GenerateLabelDialog | `generate-label-dialog.tsx` | Shipping label generation (PDF/ZPL/EPL) |
| TrackingTimeline | `tracking-timeline.tsx` | Visual shipment tracking timeline |

---

## PRODUCTS (`/components/products/`)

| Component | File | Purpose |
|-----------|------|---------|
| ProductForm | `product-form.tsx` | Product creation/editing — images (up to 16), AI descriptions, dimensions, dropship toggle |

---

## USERS (`/components/users/`)

| Component | File | Purpose |
|-----------|------|---------|
| UserForm | `user-form.tsx` | Create/edit user (email, name, role) |

---

## INVOICES (`/components/invoices/`)

| Component | File | Purpose |
|-----------|------|---------|
| InvoiceTable | `invoice-table.tsx` | DataTable for invoices with KSeF status |

---

## INTEGRATIONS (`/components/integrations/`)

| Component | File | Purpose |
|-----------|------|---------|
| IntegrationForm | `integration-form.tsx` | Dynamic credential form (20+ providers), grouped by category |
| AllegroErrorCard | `allegro-error-card.tsx` | Allegro integration error display |

---

## EDITOR (`/components/editor/`)

| Component | File | Purpose |
|-----------|------|---------|
| DescriptionEditor | `description-editor.tsx` | TipTap rich HTML editor with AI toolbar (Generate, Improve, Translate) |

---

## WORKFLOW (`/components/workflow/`)

| Component | File | Purpose |
|-----------|------|---------|
| WorkflowCanvas | `workflow-canvas.tsx` | Visual automation node canvas (pan, zoom, drag) |
| WorkflowNode | `workflow-node.tsx` | Individual node (trigger, action, condition) |
| WorkflowSidebar | `workflow-sidebar.tsx` | Node palette |
| NodeConfigPanel | `node-config-panel.tsx` | Node configuration panel |
| WorkflowToolbar | `workflow-toolbar.tsx` | Canvas tools (save, test) |

---

## ONBOARDING (`/components/onboarding/`)

| Component | File | Purpose |
|-----------|------|---------|
| OnboardingWizard | `onboarding-wizard.tsx` | Guided checklist card with progress bar |

---

## SHIPPING (`/components/shipping/`)

| Component | File | Purpose |
|-----------|------|---------|
| RateShopping | `rate-shopping.tsx` | Carrier rate comparison (postal, dimensions, weight, COD) |

---

## SHARED/REUSABLE (`/components/shared/`)

| Component | File | Purpose |
|-----------|------|---------|
| DataTable | `data-table.tsx` | Sortable/selectable table with resizable columns, density, editable cells |
| StatusBadge | `status-badge.tsx` | Colored status indicator |
| PaczkomatSelector | `paczkomat-selector.tsx` | InPost locker picker (search + geowidget map) |
| CategoryTreePicker | `category-tree-picker.tsx` | Hierarchical category selector |
| OrderSearchCombobox | `order-search-combobox.tsx` | Debounced order lookup (300ms) |
| TagInput | `tag-input.tsx` | Multi-tag input with keyboard support |
| StatusTransitionDialog | `status-transition-dialog.tsx` | Confirmation dialog for status changes |
| ProviderPicker | `provider-picker.tsx` | Grid of provider cards |
| ProviderCard | `provider-card.tsx` | Individual clickable provider card |
| CommandPalette | `command-palette.tsx` | Global Cmd+K navigation + quick actions |
| PageHeader | `page-header.tsx` | Standard page title + description + action |
| AdminGuard | `admin-guard.tsx` | Hide content from non-admin users |
| ErrorBoundary | `error-boundary.tsx` | React error boundary |
| EmptyState | `empty-state.tsx` | Consistent empty state display |
| LoadingSkeleton | `loading-skeleton.tsx` | Placeholder skeleton |
| DensityToggle | `density-toggle.tsx` | Table row density control |
| DevelopmentBanner | `development-banner.tsx` | Dev environment alert |
| EditableCell | `editable-cell.tsx` | Inline-editable table cell |
| DataTablePagination | `data-table-pagination.tsx` | Pagination controls |

---

## LAYOUT (`/components/layout/`)

| Component | File | Purpose |
|-----------|------|---------|
| Header | `header.tsx` | Top nav with breadcrumbs, theme toggle, user menu |
| Sidebar | `sidebar.tsx` | Collapsible nav sidebar with groups |
| MobileNav | `mobile-nav.tsx` | Mobile navigation drawer |
| Breadcrumbs | `breadcrumbs.tsx` | Dynamic breadcrumb trail |
| UserMenu | `user-menu.tsx` | Account menu + logout |
| ThemeToggle | `theme-toggle.tsx` | Dark/light switcher |
| ConnectionStatus | `connection-status.tsx` | Real-time server connection indicator |
| SidebarContext | `sidebar-context.tsx` | Sidebar state management |

---

## PROVIDERS (`/components/providers/`)

| Component | File | Purpose |
|-----------|------|---------|
| AuthProvider | `auth-provider.tsx` | JWT auth initialization |
| QueryProvider | `query-provider.tsx` | TanStack React Query config |
| ThemeProvider | `theme-provider.tsx` | next-themes integration |

---

## OTHER

| Component | File | Purpose |
|-----------|------|---------|
| SubscriptionBanner | `subscription-banner.tsx` | Tenant plan status banner |

---

## KEY TECHNOLOGIES

- **react-hook-form** — form state + Zod validation
- **TanStack React Query** — server state management
- **dnd-kit** — drag-and-drop (Kanban)
- **Recharts** — charting (BarChart, PieChart, AreaChart)
- **TipTap** — rich text editor
- **shadcn/ui** — base UI components
- **next-themes** — dark mode
- **Zustand** — auth store
