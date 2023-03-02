import http from 'k6/http';
import { sleep } from 'k6';

export default function () {
  http.get('https://kubeshop.github.io/testkube/');
  sleep(1);
}
