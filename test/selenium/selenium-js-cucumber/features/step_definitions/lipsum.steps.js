import { Given, Then } from '@cucumber/cucumber';
import { expect } from 'chai';

Given('I open the Lipsum page', async function () {
  await this.driver.get('https://testkube-test-page-lipsum.pages.dev/');
});

Then('the page title should be {string}', async function (expected) {
  const title = await this.driver.getTitle();
  expect(title).to.equal(expected);

  await new Promise(r => setTimeout(r, 5000)); //just for the example to last longer
});