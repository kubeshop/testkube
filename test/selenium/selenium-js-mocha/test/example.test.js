import { expect } from 'chai';
import { createDriver } from './baseTest.js';

describe('Example test 1', function () {
  this.timeout(20000);
  let driver;
  beforeEach(async () => { driver = await createDriver(); });
  afterEach(async () => { if (driver) await driver.quit(); });

  it('case 1', async () => {
    await driver.get('https://testkube-test-page-lipsum.pages.dev/');
    expect(await driver.getTitle()).to.equal('Testkube test page - Lorem Ipsum');
  });

  it('case 2', async () => {
    await new Promise(r => setTimeout(r, 500));
  });

  it('case 3', async () => {
    await new Promise(r => setTimeout(r, 2000));
  });
});