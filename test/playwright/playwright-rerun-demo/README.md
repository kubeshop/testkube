# Playwright rerun demo

Fifty Playwright tests that pass or fail randomly (about 70% pass, 30% fail). Used to demo running all tests, then re-running only the failed subset.

## Demo flow

1. **Run all 50 tests** (some will fail):
   ```bash
   npm run test:all
   # or: npx playwright test
   ```
   Playwright writes failed test info to `test-results/.last-run.json`.

2. **Re-run only the tests that failed**:
   ```bash
   npm run test:failed
   # or: npx playwright test --last-failed
   ```
   This uses `test-results/.last-run.json` to run just the failed tests from the previous run.

## Requirements

- Node and npm
- `npm install` (or `npm ci`) once
- Playwright 1.56.1 (`@playwright/test` in package.json)

## CI / Testkube

- Set `PLAYWRIGHT_OUTPUT_DIR` if you want `.last-run.json` written somewhere else (e.g. an artifacts directory).
- Persist `test-results/` (or your output dir) between the “run all” and “run failed” steps so `.last-run.json` is available for the second run.
