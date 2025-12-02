module.exports = {
  default: {
    require: ["./features/steps/**/*.js"],
    publishQuiet: true,
    format: ["progress"],
    paths: ["features/**/*.feature"]
  }
};


