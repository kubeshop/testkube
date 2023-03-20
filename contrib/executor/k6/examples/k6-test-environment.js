import http from 'k6/http';
import { sleep } from 'k6';

export default function () {
  http.get(`https://${__ENV.TARGET_HOSTNAME}/testkube/`);
  sleep(1);
}
