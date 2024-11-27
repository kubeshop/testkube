import http from 'k6/http';
import { check } from 'k6';

export default function () {
  const res = http.get('https://testkube-test-page-lipsum.pages.dev/');
  check(res, { 'status was 200': (r) => r.status == 200 });
  check(res, {
    'verify partial text': (r) =>
      r.body.includes('Testkube test page - Lipsum'),
  });
}
