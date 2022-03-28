import http from 'k6/http';
import { sleep,check } from 'k6';

export default function () {
  
  const baseURI = `${__ENV.TESTKUBE_API_URI || 'http://testkube-api-server:8088'}`;

  check(http.get(`${baseURI}/v1/info`), {
    'api server should return version': r => r.json()["version"] != undefined && r.json()["version"] != "",
    'api server should return commit': r => r.json()["commit"] != undefined && r.json()["commit"] != "",
  });


  sleep(1);
}

