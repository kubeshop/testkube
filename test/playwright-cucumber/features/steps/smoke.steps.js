// @ts-check
const { Before, After, Given, Then } = require("@cucumber/cucumber");
const { chromium, expect } = require("@playwright/test");

/** @type {import('@playwright/test').Browser} */
let browser;
/** @type {import('@playwright/test').Page} */
let page;

Before(async function () {
  if (!browser) {
    browser = await chromium.launch();
  }
  page = await browser.newPage();
});

After(async function () {
  if (page) {
    await page.close();
    page = null;
  }
  if (browser) {
    await browser.close();
    browser = null;
  }
});

Given("I open the Testkube demo page", async function () {
  await page.goto("https://testkube-test-page-lipsum.pages.dev/");
});

Then('the page title should contain "Testkube"', async function () {
  await expect(page).toHaveTitle(/Testkube/);
});


