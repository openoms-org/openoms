import { test, expect, type Page } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

/**
 * OPE-423d end-to-end validation — operations control tower: cutover
 * parity-readiness indicator + operator action wiring (resolve / retry).
 *
 * This suite is the dashboard-observable complement to the Go scenario-integration
 * tests. It mocks the read API (/v1/operations/* and /v1/fulfillment/*) with
 * page.route so the parity verdict and the operator actions render deterministically
 * without depending on recorded fulfillment data (empty in production while
 * FULFILLMENT_PROCESS_ENABLED is off). All other requests (auth, etc.) pass through
 * to the live backend. It MIRRORS fulfillment-operations.spec.ts (the OPE-419-fe
 * spec): gotoWithAuth + Polish UI labels + the same testids.
 *
 * It covers the Phase-14 dashboard-observable scenarios NOT already asserted by
 * fulfillment-operations.spec.ts:
 *   - parity-readiness indicator renders coverage % + a NOT-ready verdict + the gap
 *     metrics (the backfill->parity gate, observed by an operator),
 *   - the same indicator's READY verdict once coverage meets the threshold,
 *   - the indicator's intentional empty state on a quiet tenant,
 *   - resolve-blocker + retry-step call the CORRECT endpoints (asserted via request
 *     interception) and refresh the detail afterward.
 *
 * NOTE: this suite requires the standard e2e harness (live API at :8080 for auth +
 * a built/served dashboard). It is written to match the existing setup but was NOT
 * executed in the implementation environment, which has no live backend.
 */

const ROUTE = '/operations/fulfillment';

const PROCESS_BLOCKED = {
  id: '00000000-0000-0000-0000-0000000000d1',
  tenant_id: 't1',
  order_id: '00000000-0000-0000-0000-0000000000e1',
  aggregate_status: 'blocked',
  health_status: 'action_required',
  created_at: '2026-06-01T10:00:00Z',
  updated_at: '2026-06-01T12:00:00Z',
};

const BLOCKER = {
  id: '00000000-0000-0000-0000-0000000000f1',
  tenant_id: 't1',
  process_id: PROCESS_BLOCKED.id,
  code: 'integration_capability_missing',
  category: 'capability',
  status: 'open',
  description: 'Brak zarejestrowanego dostawcy realizacji',
  created_at: '2026-06-01T11:00:00Z',
  updated_at: '2026-06-01T11:00:00Z',
};

const FAILED_STEP = {
  id: '00000000-0000-0000-0000-0000000000f2',
  tenant_id: 't1',
  unit_id: 'u1',
  step_key: 'create_shipment',
  status: 'failed',
  attempts: 1,
  created_at: '2026-06-01T10:00:00Z',
  updated_at: '2026-06-01T12:00:00Z',
};

const DETAIL_BLOCKED = {
  process: PROCESS_BLOCKED,
  blockers: [BLOCKER],
  units: [
    {
      unit: {
        id: 'u1',
        tenant_id: 't1',
        process_id: PROCESS_BLOCKED.id,
        unit_type: 'warehouse',
        status: 'blocked',
        created_at: '2026-06-01T10:00:00Z',
        updated_at: '2026-06-01T12:00:00Z',
      },
      steps: [FAILED_STEP],
    },
  ],
  provider_attempts: [],
};

/** Common summary/exceptions/capability mocks so the page renders. */
async function mockOperationsApi(page: Page) {
  await page.route('**/v1/operations/summary', (route) =>
    route.fulfill({
      json: {
        buckets: {
          ready: 2,
          processing: 1,
          stuck: 0,
          blocked: 1,
          provider_issue: 0,
          missing_data: 0,
        },
        total: 4,
      },
    }),
  );

  await page.route('**/v1/operations/integration-capability-summary*', (route) =>
    route.fulfill({
      json: {
        automation_coverage: 0.5,
        manual_intervention_processes: 1,
        unsupported_capability_processes: 1,
        stale_data_processes: 0,
        missing_mapping_processes: 0,
        failed_provider_attempts: 0,
        total_processes: 4,
      },
    }),
  );

  await page.route('**/v1/operations/exceptions*', (route) =>
    route.fulfill({
      json: {
        items: [{ process: PROCESS_BLOCKED, bucket: 'blocked', top_blocker: BLOCKER }],
        total: 1,
      },
    }),
  );

  await page.route('**/v1/fulfillment/processes/*', (route) =>
    route.fulfill({ json: DETAIL_BLOCKED }),
  );
}

test.describe('Operations parity readiness (cutover gate)', () => {
  test('NOT ready: indicator renders coverage %, a not-ready verdict and the missing-process gap', async ({
    page,
  }) => {
    await mockOperationsApi(page);
    // Backfill->parity gate: 2 of 4 covered -> 0.5, below the threshold.
    await page.route('**/v1/operations/parity', (route) =>
      route.fulfill({
        json: {
          non_terminal_orders: 4,
          fulfillment_processes: 2,
          orders_missing_process: 2,
          process_coverage: 0.5,
          legacy_problem_orders: 1,
          process_backed_exceptions: 1,
          coverage_threshold: 0.99,
          process_coverage_met: false,
        },
      }),
    );

    await gotoWithAuth(page, ROUTE);

    const indicator = page.getByTestId('parity-readiness-indicator');
    await expect(indicator).toBeVisible({ timeout: 15000 });

    // Verdict: NOT ready.
    const verdict = indicator.getByTestId('parity-verdict');
    await expect(verdict).toBeVisible();
    await expect(verdict).toHaveAttribute('data-met', 'false');
    await expect(indicator.getByText('Brak zgodności')).toBeVisible();

    // Coverage percentage rendered (0.5 -> 50%).
    await expect(indicator.getByText('50%')).toBeVisible();
    // Threshold hint (0.99 -> 99%).
    await expect(indicator.getByText(/99%/)).toBeVisible();

    // The coverage gap is surfaced (orders missing a process).
    await expect(indicator.getByText('Zamówienia bez procesu')).toBeVisible();
    await expect(indicator.getByText('Musi osiągnąć 0 przed przełączeniem.')).toBeVisible();
  });

  test('READY: full coverage shows the ready-to-cut-over verdict', async ({ page }) => {
    await mockOperationsApi(page);
    // Post-backfill: every non-terminal order has a process -> coverage 1.0, met.
    await page.route('**/v1/operations/parity', (route) =>
      route.fulfill({
        json: {
          non_terminal_orders: 4,
          fulfillment_processes: 4,
          orders_missing_process: 0,
          process_coverage: 1.0,
          legacy_problem_orders: 0,
          process_backed_exceptions: 1,
          coverage_threshold: 0.99,
          process_coverage_met: true,
        },
      }),
    );

    await gotoWithAuth(page, ROUTE);

    const indicator = page.getByTestId('parity-readiness-indicator');
    await expect(indicator).toBeVisible({ timeout: 15000 });

    const verdict = indicator.getByTestId('parity-verdict');
    await expect(verdict).toHaveAttribute('data-met', 'true');
    await expect(indicator.getByText('Gotowe do przełączenia')).toBeVisible();
    await expect(indicator.getByText('100%')).toBeVisible();
  });

  test('EMPTY: a quiet tenant renders the intentional "nothing to compare" state', async ({
    page,
  }) => {
    await mockOperationsApi(page);
    // Vacuous full coverage on an empty tenant: the component shows the empty state,
    // not a misleading "ready" verdict.
    await page.route('**/v1/operations/parity', (route) =>
      route.fulfill({
        json: {
          non_terminal_orders: 0,
          fulfillment_processes: 0,
          orders_missing_process: 0,
          process_coverage: 1.0,
          legacy_problem_orders: 0,
          process_backed_exceptions: 0,
          coverage_threshold: 0.99,
          process_coverage_met: true,
        },
      }),
    );

    await gotoWithAuth(page, ROUTE);

    const indicator = page.getByTestId('parity-readiness-indicator');
    await expect(indicator).toBeVisible({ timeout: 15000 });
    await expect(indicator.getByText('Brak danych do porównania')).toBeVisible();
    // The verdict block is NOT rendered in the empty state.
    await expect(indicator.getByTestId('parity-verdict')).toHaveCount(0);
  });
});

test.describe('Operations operator actions (endpoint wiring)', () => {
  test.beforeEach(async ({ page }) => {
    await mockOperationsApi(page);
    await page.route('**/v1/operations/parity', (route) =>
      route.fulfill({
        json: {
          non_terminal_orders: 1,
          fulfillment_processes: 1,
          orders_missing_process: 0,
          process_coverage: 1.0,
          legacy_problem_orders: 0,
          process_backed_exceptions: 1,
          coverage_threshold: 0.99,
          process_coverage_met: true,
        },
      }),
    );
  });

  test('resolve-blocker calls POST /v1/fulfillment/blockers/{id}/resolve and refreshes the detail', async ({
    page,
  }) => {
    let resolveCalls = 0;
    let resolvedUrl = '';
    let resolveMethod = '';
    // After a successful resolve the detail is refetched; serve the resolved view.
    let detailFetches = 0;
    await page.route('**/v1/fulfillment/processes/*', (route) => {
      detailFetches += 1;
      // First fetch: open blocker. After resolve: the blocker is resolved (so the
      // panel shows "no active blockers"), proving the detail was refreshed.
      const blockers =
        detailFetches > 1 ? [{ ...BLOCKER, status: 'resolved' }] : [BLOCKER];
      return route.fulfill({ json: { ...DETAIL_BLOCKED, blockers } });
    });
    await page.route('**/v1/fulfillment/blockers/*/resolve', (route) => {
      resolveCalls += 1;
      resolvedUrl = route.request().url();
      resolveMethod = route.request().method();
      return route.fulfill({ json: { ...BLOCKER, status: 'resolved' } });
    });

    await gotoWithAuth(page, ROUTE);

    const exception = page
      .locator('[data-testid="fulfillment-exception-item"]')
      .filter({ hasText: 'Brak zarejestrowanego dostawcy realizacji' });
    await expect(exception).toBeVisible({ timeout: 15000 });
    await exception.click();

    const panel = page.getByTestId('fulfillment-detail-panel');
    await expect(panel).toBeVisible();
    await panel.getByTestId('resolve-blocker-button').click();

    // Confirm dialog -> confirm (scoped to the modal to avoid matching the panel button).
    await page.getByRole('alertdialog').getByRole('button', { name: /Rozwiąż/ }).click();

    // Success toast.
    await expect(
      page.locator('[data-sonner-toast]').filter({ hasText: /Blokada rozwiązana/ }),
    ).toBeVisible({ timeout: 10000 });

    // The CORRECT endpoint was hit (the blocker's id, /resolve, POST).
    expect(resolveCalls).toBe(1);
    expect(resolveMethod).toBe('POST');
    expect(resolvedUrl).toContain(`/v1/fulfillment/blockers/${BLOCKER.id}/resolve`);

    // The detail refreshed: the resolved blocker no longer shows as an open blocker.
    await expect(panel.getByText('Brak aktywnych blokad')).toBeVisible({
      timeout: 10000,
    });
    expect(detailFetches).toBeGreaterThan(1);
  });

  test('retry-step calls POST /v1/fulfillment/steps/{id}/retry and confirms with a toast', async ({
    page,
  }) => {
    let retryCalls = 0;
    let retriedUrl = '';
    let retryMethod = '';
    await page.route('**/v1/fulfillment/steps/*/retry', (route) => {
      retryCalls += 1;
      retriedUrl = route.request().url();
      retryMethod = route.request().method();
      return route.fulfill({
        json: { ...FAILED_STEP, status: 'pending' },
      });
    });

    await gotoWithAuth(page, ROUTE);

    const exception = page
      .locator('[data-testid="fulfillment-exception-item"]')
      .filter({ hasText: 'Brak zarejestrowanego dostawcy realizacji' });
    await expect(exception).toBeVisible({ timeout: 15000 });
    await exception.click();

    const panel = page.getByTestId('fulfillment-detail-panel');
    await expect(panel).toBeVisible();
    // The failed create_shipment step exposes a Retry action.
    await panel.getByTestId('retry-step-button').click();

    // Confirm dialog -> confirm (the retry confirm button label is "Ponów"; scope to
    // the modal so the panel's Retry button is not matched).
    await page.getByRole('alertdialog').getByRole('button', { name: /Ponów/ }).click();

    // Success toast.
    await expect(
      page
        .locator('[data-sonner-toast]')
        .filter({ hasText: /Krok skierowany do ponowienia/ }),
    ).toBeVisible({ timeout: 10000 });

    // The CORRECT endpoint was hit (the step's id, /retry, POST).
    expect(retryCalls).toBe(1);
    expect(retryMethod).toBe('POST');
    expect(retriedUrl).toContain(`/v1/fulfillment/steps/${FAILED_STEP.id}/retry`);
  });
});
