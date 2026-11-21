# OpenOMS Order Creation - Playwright E2E Automation

## Coverage
- Playwright UI automation
- Negative and edge-case testing
- Smoke and regression tagging
- Independent API testing
- PostgreSQL database validation
- Page Object Model
- GitHub Actions CI
- HTML reports and traces

## Test Structure
tests/
- order-creation.spec.js
- order-creation-negative.spec.js
- api-order.spec.js
- pom-smoke.spec.js
- auth.setup.js

pages/
- OrderCreatePage.js

helpers/
- actions.js
- db.js

## Main Flow
UI Order Creation -> Order ID -> API Validation -> Database Validation

## Run Tests
npx playwright test --project=chromium --workers=1 --retries=0

## Test Results
Negative/edge: 5/5 passed
Regression: 5/5 passed
POM smoke: 2/2 passed
Independent API: 2/2 passed
UI + API + DB showcase: passed

## CI
.github/workflows/order-creation-e2e.yml

## Open Source
Upstream PR: openoms-org/openoms#729
