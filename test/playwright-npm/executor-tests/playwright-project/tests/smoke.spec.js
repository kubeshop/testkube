// @ts-check
const { test, expect } = require("@playwright/test");

test("Smoke 1 - has title", async ({ page }) => {
  await page.goto("https://testkube-test-page-lipsum.pages.dev/");
  await page.waitForTimeout(10000)
  await expect(page).toHaveTitle(/Testkube/);
});
