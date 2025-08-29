import { setWorldConstructor, setDefaultTimeout, Before, After } from '@cucumber/cucumber';
import { Builder } from 'selenium-webdriver';

setDefaultTimeout(30000);

class World {
  async init() {
    const url = process.env.REMOTE_WEBDRIVER_URL;
    if (!url) throw new Error('REMOTE_WEBDRIVER_URL not set');
    const browser = process.env.BROWSER || 'chrome';
    this.driver = await new Builder().usingServer(url).forBrowser(browser).build();
  }
}

setWorldConstructor(World);

Before(async function () {
  await this.init();
});

After(async function () {
  if (this.driver) {
    await this.driver.quit();
  }
});