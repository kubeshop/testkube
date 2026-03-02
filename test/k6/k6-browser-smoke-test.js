import { browser } from 'k6/browser';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    ui: {
      executor: 'shared-iterations',
      vus: 1,
      iterations: 10,
      maxDuration: '30s',
      options: {
        browser: {
          type: 'chromium',
        },
      },
    },
  },
};

export default async function () {
  const context = await browser.newContext();
  const page = await context.newPage();

  const res = await page.goto('https://testkube-test-page-lipsum.pages.dev/');

  check(res, {
    'status is 200': (r) => r.status() === 200,
  });

  const title = await page.title();

  check(title, {
    'title is correct': (t) => t === 'Testkube test page - Lorem Ipsum',
  });

  await page.waitForTimeout(1000);
  await context.close();
}