import http from 'k6/http';
import { check } from 'k6';

export default function () {
  const baseURI = `${__ENV.API_URI || 'http://chuck-jokes.services:8881'}`;

  check(http.get(`${baseURI}/joke`), {
    'joke should be about Chuck': r => r.body.includes("Chuck")
  });
}