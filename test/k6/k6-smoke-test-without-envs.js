import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  thresholds: {
    "http_req_duration{scenario:default}": ["p(95)<2000"],
    "http_reqs{scenario:default}": ["count>0"],
    "http_req_failed{scenario:default}": ["rate<0.5"],
  },
};

export default function () {
  const res = http.get("https://testkube-test-page-lipsum.pages.dev/");
  check(res, {
    "status is 200": (r) => r.status === 200,
  });
  sleep(1);
}
