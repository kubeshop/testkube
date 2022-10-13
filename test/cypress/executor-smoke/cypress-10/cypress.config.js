const { defineConfig } = require("cypress");

module.exports = defineConfig({
  e2e: {
    baseUrl: 'https://testkube.kubeshop.io/',
    video: false
  },
});
