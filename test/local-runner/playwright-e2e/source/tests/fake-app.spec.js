const { test, expect } = require('@playwright/test');
const fs = require('node:fs/promises');

const baseURL = process.env.BASE_URL || 'http://local-playwright-app.testkube-local.svc.cluster.local:8080';

test('runs a browser interaction against the in-cluster fake app', async ({ page }, testInfo) => {
  await page.goto(baseURL);

  await expect(page).toHaveTitle('Local TestWorkflow Playwright Demo');
  await expect(page.getByRole('heading', { name: 'Local TestWorkflow browser check' })).toBeVisible();
  await expect(page.locator('#count')).toHaveText('0');

  await page.getByRole('button', { name: 'Increment' }).click();
  const counter = await page.locator('#count').textContent();

  await page.screenshot({
    path: testInfo.outputPath('after-increment.png'),
    fullPage: true,
  });
  await fs.writeFile(
    testInfo.outputPath('direct-local-proof.json'),
    `${JSON.stringify({
      baseURL,
      counter,
      marker: 'created-in-direct-local-job',
    }, null, 2)}\n`,
  );

  await expect(page.locator('#count')).toHaveText(process.env.EXPECTED_COUNTER || '1');
});
