const { test, expect } = require('@playwright/test');

const baseURL = process.env.BASE_URL || 'http://local-playwright-app.testkube-local.svc.cluster.local:8080';

test('runs a browser interaction against the in-cluster fake app', async ({ page }) => {
  await page.goto(baseURL);

  await expect(page).toHaveTitle('Local TestWorkflow Playwright Demo');
  await expect(page.getByRole('heading', { name: 'Local TestWorkflow browser check' })).toBeVisible();
  await expect(page.locator('#count')).toHaveText('0');

  await page.getByRole('button', { name: 'Increment' }).click();
  await expect(page.locator('#count')).toHaveText('1');
});
