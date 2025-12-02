import { setWorldConstructor, setDefaultTimeout, Before, After } from "@cucumber/cucumber";
import { chromium } from "playwright";

setDefaultTimeout(60 * 1000);

class PlaywrightWorld {
  constructor() {
    /** @type {import('playwright').Browser} */
    this.browser = null;
    /** @type {import('playwright').Page} */
    this.page = null;
  }

  async openBrowser() {
    if (!this.browser) {
      this.browser = await chromium.launch();
    }
  }

  async newPage() {
    await this.openBrowser();
    this.page = await this.browser.newPage();
    return this.page;
  }

  async closeBrowser() {
    if (this.browser) {
      await this.browser.close();
      this.browser = null;
      this.page = null;
    }
  }
}

setWorldConstructor(PlaywrightWorld);

Before(async function () {
  await this.openBrowser();
});

After(async function () {
  await this.closeBrowser();
});


