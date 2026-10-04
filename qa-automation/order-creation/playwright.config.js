const { defineConfig, devices } = require('@playwright/test');
const path = require('path');

const authFile = path.resolve(__dirname, '.auth/user.json');

module.exports = defineConfig({
  testDir: './tests',

  timeout: 60000,

  expect: {
    timeout: 15000,
  },

  workers: 1,
  retries: 0,

  use: {
    baseURL: 'http://localhost:3000',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },

  projects: [
    {
      name: 'setup',
      testMatch: /auth\.setup\.js/,
    },
    {
      name: 'chromium',
      dependencies: ['setup'],
      use: {
        ...devices['Desktop Chrome'],
      },
    },
  ],

  reporter: [
    ['list'],
    ['html', {
      outputFolder: 'playwright-report',
      open: 'never',
    }],
  ],
});

