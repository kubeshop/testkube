import http from 'k6/http';
import { sleep,check } from 'k6';

export const options = {
    stages: [
      { duration: '1m', target: 1000 },
      { duration: '10m', target: 1000 },
      { duration: '1m', target: 0 },
    ],
  };

export default function () {
  
  const baseURI = 'http://34.111.130.124';

  const responses = http.batch([

    ['GET', `${baseURI}/1`, null, { tags: { name: '1' } }],
    ['GET', `${baseURI}/2`, null, { tags: { name: '2' } }],
  ]);
}

