// @ts-check
const { test, expect } = require('@playwright/test');

test('has title', async ({ page }) => {
  await page.goto('https://testkube.io/');

  await expect(page).toHaveTitle(/ASDFASDF/);
});