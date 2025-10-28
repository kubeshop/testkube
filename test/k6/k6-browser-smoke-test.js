import { browser } from 'k6/experimental/browser';
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
  const page = browser.newPage();

  try {
    await page.goto('https://testkube-test-page-lipsum.pages.dev/');
    await page.waitForTimeout(3000); // increase test duration

    const title = await page.title();

    check(title, {
      'title contains "Testkube"': (t) => /Testkube/.test(t),
    });
  } finally {
    page.close();
  }
}