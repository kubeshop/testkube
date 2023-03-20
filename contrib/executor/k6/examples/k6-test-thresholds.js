import http from 'k6/http';
import { sleep, check } from 'k6';

export let options = {
  insecureSkipTLSVerify: true,
  thresholds: {
      'http_req_duration{kind:html}': ['avg<=250', 'p(95)<500'],
      'checks{kind:http}': ['rate>0.95'],     // example for tag specific checks
      checks: ['rate>0.95'],                  // example for overall checks
  },
};

export default function () {
  check(http.get('https://kubeshop.github.io/testkube/', {
      tags: {kind: 'html'},
  }), {
      "status is 404": (res) => res.status === 404,
  }, {kind: 'http'},
  );
  sleep(1);
}