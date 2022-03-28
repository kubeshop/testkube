import http from 'k6/http';
import { sleep,check } from 'k6';

export default function () {
  const baseURI = `${__ENV.TESTKUBE_HOMEPAGE_URI || 'https://testkube.kubeshop.io'}`
  check(http.get(`${baseURI}/`), {
    'check testkube homepage home page': (r) =>
      r.body.includes('Your friendly cloud-native testing framework for Kubernetes'),
  });


  sleep(1);
}
