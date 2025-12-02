const reporter = require('cucumber-html-reporter');
const fs = require('fs');
const path = require('path');

// Ensure reports directory exists
const reportsDir = 'reports';
if (!fs.existsSync(reportsDir)) {
    fs.mkdirSync(reportsDir, { recursive: true });
}

const jsonReportPath = path.join(reportsDir, 'cucumber_report.json');

// Check if JSON report exists
if (!fs.existsSync(jsonReportPath)) {
    console.error(`Error: JSON report file not found at ${jsonReportPath}`);
    console.error('Please run "npm test" first to generate the report.');
    process.exit(1);
}

const options = {
    theme: 'bootstrap',
    jsonFile: jsonReportPath,
    output: path.join(reportsDir, 'cucumber_report.html'),
    reportSuiteAsScenarios: true,
    scenarioTimestamp: true,
    launchReport: false, // Set to false for CI/Docker environments
    metadata: {
        "App Version": "1.0.0",
        "Test Environment": "STAGING",
        "Browser": "Chrome",
        "Platform": "Docker"
    }
};

reporter.generate(options);