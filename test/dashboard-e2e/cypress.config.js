const { defineConfig } = require("cypress");

module.exports = defineConfig({
  e2e: {
    setupNodeEvents(on, config) {
      // implement node event listeners here
    },
    defaultCommandTimeout: 10000,
    baseUrl: 'http://localhost:8080',
    env: {
      API_URL: 'http://localhost:8088/v1'
    },
  },
});
