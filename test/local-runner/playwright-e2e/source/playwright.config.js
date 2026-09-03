const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  reporter: 'line',
  use: {
    actionTimeout: 0,
    trace: 'retain-on-failure',
    ...devices['Desktop Chrome'],
  },
});
