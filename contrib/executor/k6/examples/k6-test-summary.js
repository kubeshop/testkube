import http from 'k6/http';
import { sleep, check } from 'k6';

export let options = {
  insecureSkipTLSVerify: true,
  thresholds: {
      'http_req_duration{kind:html}': ['avg<=250', 'p(95)<500'],
  }
};

export default function () {
  check(http.get('https://kubeshop.github.io/testkube/', {
      tags: {'kind': 'html'},
  }), {
      "status is 200": (res) => res.status === 200,
  });
  sleep(1);
}

import { jUnit, textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';

export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: false }),
    'junit.xml': jUnit(data), // but also transform it and save it as a JUnit XML...
    'summary.json': JSON.stringify(data), // and a JSON with all the details...
  };
}
