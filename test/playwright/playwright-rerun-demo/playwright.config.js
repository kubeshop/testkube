const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  timeout: 30 * 1000,
  expect: {
    timeout: 5000
  },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  // No retries: failed tests are recorded in test-results/.last-run.json for --last-failed re-run demo.
  retries: 0,
  workers: process.env.CI ? 1 : undefined,
  // Where Playwright writes .last-run.json (used by npx playwright test --last-failed).
  outputDir: process.env.PLAYWRIGHT_OUTPUT_DIR || 'test-results',
  reporter: process.env.CI ? [['list'], ['junit', { outputFile: process.env.PLAYWRIGHT_JUNIT_OUTPUT_NAME || 'junit-report.xml' }], ['html', { open: 'never', outputFolder: process.env.PLAYWRIGHT_HTML_REPORT || 'playwright-report' }]] : 'html',
  use: {
    actionTimeout: 0,
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
