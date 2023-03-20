import http from 'k6/http';
import { check } from 'k6';

export default function () {
  const baseURI = `${__ENV.API_URI || 'http://google.pl'}`;

  check(http.get(`${baseURI}/joke`), {
    'joke should be about Chuck': r => r.body.includes("Chuck") // this should fail
  });
}
