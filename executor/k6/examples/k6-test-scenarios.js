import http from 'k6/http';
import { sleep, check } from 'k6';

export let options = {
  insecureSkipTLSVerify: true,
  thresholds: {
      'http_req_duration{kind:html}': ['avg<=250', 'p(95)<500'],
      'checks{type:testkube}': ['rate>0.95'],
      'checks{type:monokle}': ['rate>0.4'],
      // checks: ['rate>0.5'],
  },
  scenarios: {
    testkube: {
      executor: 'constant-vus',
      exec: 'testkube',
      vus: 5,
      duration: '10s',
      tags: { type: 'testkube' },
    },
    monokle: {
      executor: 'per-vu-iterations',
      exec: 'monokle',
      vus: 5,
      iterations: 10,
      startTime: '5s',
      maxDuration: '1m',
      tags: { type: 'monokle' },
    },
  },
};

export function testkube() {
  check(http.get('https://kubeshop.github.io/testkube/', {
      tags: {'kind': 'html'},
  }), {
      "Testkube is OK": (res) => res.status === 200,
  });
  sleep(1);
}

export function monokle() {
  check(http.get('https://kubeshop.github.io/monokle/', {
      tags: {'kind': 'html'},
  }), {
      "Monokle is OK": (res) => res.status === 200,
  });
  sleep(1);
}

