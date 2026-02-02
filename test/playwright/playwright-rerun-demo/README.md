# Playwright rerun demo

Fifty Playwright tests that pass or fail randomly (about 70% pass, 30% fail). Used to demo running all tests, then re-running only the failed subset.

## Demo flow

1. **Go to Testkube:** 
   ```bash
   https://app.testkube.io/organization/testkube-internal-demo/environment/paris/dashboard/test-workflows/playwright-rerun-demo/overview
   ```

   See reference docs: https://docs.testkube.io/articles/examples/playwright-rerun

2. **Generate failed tests:**
   Run the workflow as is - this will generate a test execution with about 15 failed test cases

3. **Rerun the workflow:**
   Run the workflow again with 'rerunFailed' flag set to true.

   Thats it. 


