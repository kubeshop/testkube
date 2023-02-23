import http from 'k6/http';
import { check } from 'k6';

export default function () {
  http.get('https://testkube.kubeshop.io/');
}