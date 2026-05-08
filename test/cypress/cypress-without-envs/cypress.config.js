const { defineConfig } = require("cypress");

module.exports = defineConfig({
  e2e: {
    baseUrl: "https://testkube-test-page-lipsum.pages.dev/",
    video: false,
  },
});
