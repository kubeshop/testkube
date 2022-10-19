import http from 'k6/http';
import { check } from 'k6';

export default function () {
  http.get('https://testkube.kubeshop.io/');

  check(__ENV.K6_ENV_FROM_PARAM, {
    'Correct ENV value is set with -e param (K6_ENV_FROM_PARAM)': (value) => value == "K6_ENV_FROM_PARAM_value",
  });
  check(__ENV.K6_SYSTEM_ENV, {
    'Correct ENV value is set from system ENV (K6_SYSTEM_ENV)': (value) => value == "K6_SYSTEM_ENV_value",
  });
  // Check results are visible in K6 summary, but non-zero exit codes aren't returned on failed checks
}