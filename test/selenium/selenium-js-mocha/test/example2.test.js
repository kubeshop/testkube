import { expect } from 'chai';
import { createDriver } from './baseTest.js';

describe('Example test 2', function () {
  this.timeout(20000);
  let driver;
  beforeEach(async () => { driver = await createDriver(); });
  afterEach(async () => { if (driver) await driver.quit(); });

  it('case 1', async () => {
    await new Promise(r => setTimeout(r, 2000));
  });

  it('case 2', async () => {
    await new Promise(r => setTimeout(r, 5000));
  });

  it('case 3', async () => {
    await new Promise(r => setTimeout(r, 3000));
  });
});