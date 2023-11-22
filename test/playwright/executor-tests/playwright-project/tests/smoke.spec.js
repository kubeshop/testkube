// @ts-check
const { test, expect } = require('@playwright/test');

test('Smoke 1 - has title', async ({ page }) => {
  await page.goto('https://testkube.io/');

  await expect(page).toHaveTitle(/Testkube/);
});