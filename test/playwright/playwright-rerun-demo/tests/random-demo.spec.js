// @ts-check
// 50 individual tests for demo: first run, then re-run only failed via --last-failed.
// Each test passes with 70% probability, fails with 30%.
const { test, expect } = require('@playwright/test');

const PASS_CHANCE = 0.7;

function shouldPass() {
  return Math.random() < PASS_CHANCE;
}

test('random-demo-1', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-2', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-3', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-4', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-5', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-6', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-7', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-8', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-9', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-10', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-11', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-12', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-13', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-14', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-15', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-16', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-17', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-18', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-19', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-20', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-21', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-22', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-23', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-24', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-25', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-26', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-27', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-28', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-29', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-30', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-31', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-32', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-33', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-34', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-35', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-36', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-37', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-38', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-39', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-40', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-41', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-42', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-43', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-44', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-45', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-46', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-47', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-48', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-49', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
test('random-demo-50', async () => {
  if (shouldPass()) expect(1).toBe(1);
  else expect(1).toBe(2);
});
