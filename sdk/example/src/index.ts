import * as testkube from "@testkube/sdk";

testkube.initTestkube({
  url: "https://demo.testkube.io/results/v1",
});

const main = async () => {
  console.log("Test started");

  const result1 = testkube.test("simple-curl").runUntilEnd();
  const result2 = testkube.test("simple-curl-2").runUntilEnd();

  const result = await Promise.all([result1, result2]);
  console.log("🚀 ~ file: index.ts:5 ~ main ~ result:", result);
};

main();
