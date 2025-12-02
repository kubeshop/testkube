module.exports = {
  default: {
    require: ["./features/steps/**/*.js"],
    publishQuiet: true,
    format: [
      "progress",
      "json:reports/cucumber_report.json"
    ],
    paths: ["features/**/*.feature"]
  }
};


