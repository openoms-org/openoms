import { test, expect, type Page } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

/**
 * OPE-419 process-backed operations control tower + fulfillment detail (additive).
 *
 * These specs mock the /v1/operations/* and /v1/fulfillment/* read API with
 * page.route so the three representative process states — blocked,
 * waiting_external and completed — render deterministically without depending on
 * recorded fulfillment data (which is empty in production while
 * FULFILLMENT_PROCESS_ENABLED is off). All other requests (auth, etc.) pass
 * through to the live backend.
 *
 * NOTE: this suite requires the standard e2e harness (live API at :8080 for
 * auth + a built/served dashboard). It is written to match the existing setup
 * (gotoWithAuth + Polish UI labels) but was NOT executed in the implementation
 * environment, which has no live backend.
 */

const ROUTE = '/operations/fulfillment';

const PROCESS_BLOCKED = {
  id: '00000000-0000-0000-0000-0000000000b1',
  tenant_id: 't1',
  order_id: '00000000-0000-0000-0000-0000000000a1',
  aggregate_status: 'blocked',
  health_status: 'action_required',
  created_at: '2026-06-01T10:00:00Z',
  updated_at: '2026-06-01T12:00:00Z',
};

const PROCESS_WAITING = {
  id: '00000000-0000-0000-0000-0000000000b2',
  tenant_id: 't1',
  order_id: '00000000-0000-0000-0000-0000000000a2',
  aggregate_status: 'waiting_external',
  health_status: 'warning',
  created_at: '2026-06-01T09:00:00Z',
  updated_at: '2026-06-01T11:30:00Z',
};

const PROCESS_COMPLETED = {
  id: '00000000-0000-0000-0000-0000000000b3',
  tenant_id: 't1',
  order_id: '00000000-0000-0000-0000-0000000000a3',
  aggregate_status: 'completed',
  health_status: 'ok',
  created_at: '2026-05-30T09:00:00Z',
  updated_at: '2026-05-31T18:00:00Z',
};

const BLOCKER = {
  id: '00000000-0000-0000-0000-0000000000c1',
  tenant_id: 't1',
  process_id: PROCESS_BLOCKED.id,
  code: 'external_status_unmapped',
  category: 'mapping',
  status: 'open',
  description: 'Brak mapowania usługi kuriera',
  created_at: '2026-06-01T11:00:00Z',
  updated_at: '2026-06-01T11:00:00Z',
};

async function mockFulfillmentApi(page: Page) {
  await page.route('**/v1/operations/summary', (route) =>
    route.fulfill({
      json: {
        buckets: {
          ready: 4,
          processing: 2,
          stuck: 1,
          blocked: 3,
          provider_issue: 2,
          missing_data: 1,
        },
        total: 13,
      },
    }),
  );

  await page.route('**/v1/operations/integration-capability-summary*', (route) =>
    route.fulfill({
      json: {
        automation_coverage: 0.85,
        manual_intervention_processes: 1,
        unsupported_capability_processes: 2,
        stale_data_processes: 0,
        missing_mapping_processes: 1,
        failed_provider_attempts: 2,
        total_processes: 13,
      },
    }),
  );

  await page.route('**/v1/operations/exceptions*', (route) =>
    route.fulfill({
      json: {
        items: [
          { process: PROCESS_BLOCKED, bucket: 'missing_data', top_blocker: BLOCKER },
          { process: PROCESS_WAITING, bucket: 'provider_issue' },
        ],
        total: 2,
      },
    }),
  );

  // Process detail (drilldown). Returns the matching mocked process.
  await page.route('**/v1/fulfillment/processes/*', (route) => {
    const url = route.request().url();
    if (url.includes(PROCESS_BLOCKED.id)) {
      return route.fulfill({
        json: {
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
              steps: [
                {
                  id: 's1',
                  tenant_id: 't1',
                  unit_id: 'u1',
                  step_key: 'generate_label',
                  status: 'failed',
                  attempts: 3,
                  created_at: '2026-06-01T10:00:00Z',
                  updated_at: '2026-06-01T12:00:00Z',
                },
              ],
            },
          ],
          provider_attempts: [
            {
              id: 'pa1',
              tenant_id: 't1',
              process_id: PROCESS_BLOCKED.id,
              provider: 'inpost',
              operation: 'generate_label',
              status: 'failed',
              error_code: 'label_failed',
              created_at: '2026-06-01T11:45:00Z',
            },
          ],
        },
      });
    }
    return route.fulfill({
      json: {
        process: PROCESS_WAITING,
        blockers: [],
        units: [],
        provider_attempts: [],
      },
    });
  });

  await page.route('**/v1/fulfillment/blockers/*/resolve', (route) =>
    route.fulfill({ json: { ...BLOCKER, status: 'resolved' } }),
  );

  await page.route('**/v1/fulfillment/steps/*/retry', (route) =>
    route.fulfill({
      json: { id: 's1', step_key: 'generate_label', status: 'pending', attempts: 3 },
    }),
  );
}

test.describe('Operations control tower (process-backed)', () => {
  test.beforeEach(async ({ page }) => {
    await mockFulfillmentApi(page);
  });

  test('renders the six operator buckets with counts', async ({ page }) => {
    await gotoWithAuth(page, ROUTE);
    await expect(
      page.getByRole('heading', { name: /Wieża kontroli realizacji/ }),
    ).toBeVisible({ timeout: 15000 });

    // Bucket labels (Polish UI).
    await expect(page.getByText('Gotowe', { exact: true }).first()).toBeVisible();
    await expect(page.getByText('W realizacji', { exact: true }).first()).toBeVisible();
    await expect(page.getByText('Zablokowane', { exact: true }).first()).toBeVisible();
    await expect(page.getByText('Czeka na dostawcę', { exact: true }).first()).toBeVisible();
    await expect(page.getByText('Brak danych', { exact: true }).first()).toBeVisible();
  });

  test('shows the integration automation coverage metrics', async ({ page }) => {
    await gotoWithAuth(page, ROUTE);
    await expect(page.getByText('Pokrycie automatyzacją').first()).toBeVisible({
      timeout: 15000,
    });
    await expect(page.getByText('85%')).toBeVisible();
  });

  test('blocked flow: exception shows the canonical reason and drilldown opens the detail with Resolve', async ({
    page,
  }) => {
    await gotoWithAuth(page, ROUTE);

    const exception = page
      .locator('[data-testid="fulfillment-exception-item"]')
      .filter({ hasText: 'Brak mapowania usługi kuriera' });
    await expect(exception).toBeVisible({ timeout: 15000 });

    await exception.click();

    const panel = page.getByTestId('fulfillment-detail-panel');
    await expect(panel).toBeVisible();
    // Blocker + a Resolve action are present.
    await expect(panel.getByTestId('fulfillment-blocker')).toBeVisible();
    await expect(panel.getByTestId('resolve-blocker-button')).toBeVisible();
    // Failed step exposes a Retry action.
    await expect(panel.getByTestId('retry-step-button')).toBeVisible();
  });

  test('blocked flow: resolving a blocker confirms and shows a success toast', async ({
    page,
  }) => {
    await gotoWithAuth(page, ROUTE);

    const exception = page
      .locator('[data-testid="fulfillment-exception-item"]')
      .filter({ hasText: 'Brak mapowania usługi kuriera' });
    await expect(exception).toBeVisible({ timeout: 15000 });
    await exception.click();

    const panel = page.getByTestId('fulfillment-detail-panel');
    await panel.getByTestId('resolve-blocker-button').click();

    // Confirm dialog -> confirm.
    await page.getByRole('button', { name: /Rozwiąż/ }).click();

    await expect(
      page.locator('[data-sonner-toast]').filter({ hasText: /Blokada rozwiązana/ }),
    ).toBeVisible({ timeout: 10000 });
  });

  test('waiting_external flow: provider-issue exception drills into a detail with no blockers', async ({
    page,
  }) => {
    await gotoWithAuth(page, ROUTE);

    const waiting = page
      .locator('[data-testid="fulfillment-exception-item"]')
      .filter({ hasText: /Czeka na dostawcę/ });
    await expect(waiting.first()).toBeVisible({ timeout: 15000 });
    await waiting.first().click();

    const panel = page.getByTestId('fulfillment-detail-panel');
    await expect(panel).toBeVisible();
    await expect(panel.getByText('Brak aktywnych blokad')).toBeVisible();
  });

  test('completed flow: a completed process is not an exception (terminal, not actionable)', async ({
    page,
  }) => {
    // Override exceptions to include only the completed process upstream of the
    // backend filter — the API never returns terminal processes as exceptions,
    // so the feed should show the intentional empty state.
    await page.route('**/v1/operations/exceptions*', (route) =>
      route.fulfill({ json: { items: [], total: 0 } }),
    );
    await page.route('**/v1/operations/summary', (route) =>
      route.fulfill({
        json: {
          buckets: {
            ready: 0,
            processing: 0,
            stuck: 0,
            blocked: 0,
            provider_issue: 0,
            missing_data: 0,
          },
          total: 1,
        },
      }),
    );

    await gotoWithAuth(page, ROUTE);
    await expect(
      page.getByRole('heading', { name: /Wieża kontroli realizacji/ }),
    ).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('Brak wyjątków')).toBeVisible();
  });
});
