const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  outputDir: 'test-results',
  reporter: [
    ['line'],
    ['junit', { outputFile: 'test-results/junit.xml' }],
    ['html', { outputFolder: 'playwright-report', open: 'never' }],
  ],
  use: {
    actionTimeout: 0,
    trace: 'on',
    screenshot: 'on',
    video: 'on',
    ...devices['Desktop Chrome'],
  },
});
