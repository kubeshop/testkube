const path = require('path');
const { Pact, Matchers } = require('@pact-foundation/pact');
const superagent = require('superagent');

const { like } = Matchers;

const provider = new Pact({
  consumer: 'UserConsumer',
  provider: 'UserProvider',
  port: 1234,
  logLevel: 'warn',
  dir: path.resolve(__dirname, '../../pacts'),
});

describe('Pact test for admin user', () => {
  beforeAll(() => provider.setup());
  afterAll(() => provider.finalize());
  afterEach(() => provider.verify());

  it('should return admin user with ID 1', async () => {
    await provider.addInteraction({
      state: 'admin user with ID 1 exists',
      uponReceiving: 'a request for user 1',
      withRequest: {
        method: 'GET',
        path: '/user/1',
      },
      willRespondWith: {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: like({
          id: 1,
          name: 'Example Admin',
          role: 'admin',
        }),
      },
    });

    const res = await superagent.get('http://localhost:1234/user/1');

    expect(res.status).toBe(200);
    expect(res.body).toEqual({
      id: 1,
      name: 'Example Admin',
      role: 'admin',
    });
  });
});
