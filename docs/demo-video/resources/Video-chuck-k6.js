
import http from 'k6/http';
import { sleep,check } from 'k6';

export default function () {
  const baseURI = `${__ENV.API_URI || 'http://testkube-api-server:8881'}`;

  check(http.get(`${baseURI}/joke`), {
    'joke should be about Chuck': r => r.body.includes("Chuck")
  });
}
