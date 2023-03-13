import http from 'k6/http';
import { check } from 'k6';

if (__ENV.K6_ENV_FROM_PARAM != "K6_ENV_FROM_PARAM_value") {
  throw new Error("Incorrect K6_ENV_FROM_PARAM ENV value");
}

if (__ENV.K6_SYSTEM_ENV != "K6_SYSTEM_ENV_value") {
  throw new Error("Incorrect K6_SYSTEM_ENV ENV value");
}

export default function () {
  http.get('https://testkube.kubeshop.io/');
}