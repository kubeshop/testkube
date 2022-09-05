import http from 'k6/http';
import { sleep,check } from 'k6';

export const options = {
    stages: [
      { duration: '30s', target: 200 },
      { duration: '5m', target: 200 },
      { duration: '10s', target: 0 },
    ],
  };

export default function () {
  
  const baseURIasdasdfasdfasdf = 'http://34.111.130.124';
  // const ipsum = 'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Praesent vitae ultricies arcu. Quisque eget orci sagittis, ullamcorper elit nec, porta nulla. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia curae; Morbi lacinia ullamcorper felis, ac cursus nibh tempus fermentum. Praesent commodo nunc dui. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Aenean eu tincidunt purus, vel faucibus ligula. Integer pellentesque elementum quam, vitae dignissim est feugiat in. Aliquam sit amet semper dolor, sit amet cursus leo. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Duis in sodales nulla.'
  // console.log(`\n ${ipsum}\n`)
  // for (let i = 0; i < 1000; i+=1) {
  //   console.log(`i: ${i}: \n ${ipsum}\n`)
  // }

  const responses = http.batch([

    ['GET', `${baseURI}/1`, null, { tags: { name: '1' } }],
    ['GET', `${baseURI}/2`, null, { tags: { name: '2' } }],
  ]);
}

