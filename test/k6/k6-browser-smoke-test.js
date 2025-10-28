import { browser } from 'k6/browser';
import { check } from 'k6';

export const options = {
  scenarios: {
    ui: {
      executor: 'shared-iterations',
      options: {
        browser: {
          type: 'chromium',
        },
      },
    },
  },
};

export default async function () {
  const context = browser.newContext();
  const page = context.newPage();

  await page.goto('https://testkube-test-page-lipsum.pages.dev/');
  await page.waitForTimeout(3000);

  const title = await page.title();

  check(title, {
    'Validate page title': (t) => t === 'Testkube test page - Lorem Ipsum',
  });

  await context.close();
}
