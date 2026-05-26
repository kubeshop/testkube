import http from "k6/http";
import { check, sleep } from "k6";

export default function () {
  const res = http.get("https://testkube-test-page-lipsum.pages.dev/");
  check(res, {
    "status is 200": (r) => r.status === 200,
  });
  sleep(1);
}