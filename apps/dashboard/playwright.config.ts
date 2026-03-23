import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 2,
  workers: 1,
  reporter: process.env.CI ? 'github' : 'html',
  timeout: 30_000,
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    actionTimeout: 10_000,
    navigationTimeout: 15_000,
  },
  projects: [
    { name: 'setup', testMatch: /.*\.setup\.ts/ },
    {
      name: 'logged-out',
      testMatch: ['auth.spec.ts', 'public-return.spec.ts'],
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'trial',
      testMatch: ['trial-flow.spec.ts'],
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        storageState: 'e2e/.auth/user.json',
      },
      dependencies: ['setup'],
      testIgnore: ['auth.spec.ts', 'public-return.spec.ts', 'trial-flow.spec.ts'],
    },
  ],
  webServer: {
    command: process.env.CI
      ? 'npm run build && cp -r .next/static .next/standalone/.next/static && cp -r public .next/standalone/public && PORT=3000 node .next/standalone/server.js'
      : 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
  },
});
