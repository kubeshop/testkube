import http from 'k6/http';
import { check } from 'k6';

export const options = {
  discardResponseBodies: true,
};

export default function () {
  const res = http.get('https://storage.googleapis.com/perf-test-static-page-bucket/testkube-test-page-lorem-ipsum/index.html');
  check(res, { 'status was 200': (r) => r.status == 200 });
}
