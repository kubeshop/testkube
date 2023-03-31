package k6detector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const validK6Script = `import http from 'k6/http';
import { sleep, check } from 'k6';
import { Counter } from 'k6/metrics';

// A simple counter for http requests

export const requests = new Counter('http_reqs');

// you can specify stages of your test (ramp up/down patterns) through the options object
// target is the number of VUs you are aiming for

export const options = {
  stages: [
    { target: 20, duration: '1m' },
    { target: 15, duration: '1m' },
    { target: 0, duration: '1m' },
  ],
  thresholds: {
    requests: ['count < 100'],
  },
};

export default function () {
  // our HTTP request, note that we are saving the response to res, which can be accessed later

  const res = http.get('http://test.k6.io');

  sleep(1);

  const checkRes = check(res, {
    'status is 200': (r) => r.status === 200,
    'response body': (r) => r.body.indexOf('Feel free to browse') !== -1,
  });
}`

const invalidK6Script = `describe('The Home Page', () => {
	it('Go to dashboard', () => {
	  cy.visit('https://demo.testkube.io/apiEndpoint?apiEndpoint=https://demo.testkube.io/results/v1');
	});
  
	it('Test suites should be shown as default view', () => {
	  cy.get('h1').should('have.text', 'Test Suites ');
	});
  });`

func TestK6Adapter(t *testing.T) {

	t.Run("detect valid k6 script", func(t *testing.T) {
		// given
		a := Detector{}

		// when
		name, ok := a.Is(apiClient.UpsertTestOptions{
			Content: testkube.NewStringTestContent(validK6Script),
		})

		// then
		assert.True(t, ok, "K6Adapter should detect valid script")
		assert.Equal(t, "k6/script", name)
	})

	t.Run("ignore invalid k6 script", func(t *testing.T) {
		// given
		a := Detector{}

		// when
		name, ok := a.Is(apiClient.UpsertTestOptions{
			Content: testkube.NewStringTestContent(invalidK6Script),
		})

		// then
		assert.False(t, ok, "K6Adapter should detect valid script")
		assert.Empty(t, name)
	})
}
