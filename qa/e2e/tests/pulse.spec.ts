import { test, expect } from '@playwright/test';

const EMAIL = `e2e${Date.now()}@pulse.local`;
const PASS = 'e2e12345';

test.beforeEach(async ({ page }) => {
  const email = `e2e${Date.now()}${Math.floor(Math.random() * 1e6)}@pulse.local`;
  await page.goto('/');
  await page.getByRole('button', { name: /register/i }).click();
  await page.getByPlaceholder('email').fill(email);
  await page.getByPlaceholder(/password/i).fill(PASS);
  await page.getByPlaceholder('name').fill('E2E');
  await page.getByRole('button', { name: /^register$/i }).click();
  await expect(page.getByText('monitors', { exact: false }).first()).toBeVisible({ timeout: 10000 });
});

test('dashboard shows summary stats', async ({ page }) => {
  await expect(page.getByText('uptime 24h')).toBeVisible();
});

test('create monitor appears in list', async ({ page }) => {
  const name = 'E2E Monitor ' + Date.now();
  await page.getByRole('button', { name: '+ New' }).click();
  await page.getByPlaceholder(/name/i).fill(name);
  await page.getByPlaceholder(/url/i).fill('https://example.com');
  await page.getByRole('button', { name: 'Create monitor' }).click();
  await page.getByRole('button', { name: 'Monitors' }).click();
  await expect(page.getByText(name)).toBeVisible({ timeout: 10000 });
});

test('create monitor rejects bad url', async ({ page }) => {
  await page.getByRole('button', { name: '+ New' }).click();
  await page.getByPlaceholder(/name/i).fill('Bad');
  await page.getByPlaceholder(/url/i).fill('http://localhost:9/x');
  await page.getByRole('button', { name: 'Create monitor' }).click();
  await expect(page.getByText(/not allowed|valid http/i)).toBeVisible({ timeout: 5000 });
});

test('monitor detail shows checks table', async ({ page }) => {
  const name = 'E2E Detail ' + Date.now();
  await page.getByRole('button', { name: '+ New' }).click();
  await page.getByPlaceholder(/name/i).fill(name);
  await page.getByPlaceholder(/url/i).fill('https://example.com');
  await page.getByRole('button', { name: 'Create monitor' }).click();
  await page.getByRole('button', { name: 'Monitors' }).click();
  await page.getByText(name).click();
  await expect(page.getByText('Recent checks')).toBeVisible({ timeout: 10000 });
});

test('pause flips monitor to paused', async ({ page }) => {
  const name = 'E2E Pause ' + Date.now();
  await page.getByRole('button', { name: '+ New' }).click();
  await page.getByPlaceholder(/name/i).fill(name);
  await page.getByPlaceholder(/url/i).fill('https://example.com');
  await page.getByRole('button', { name: 'Create monitor' }).click();
  await page.getByRole('button', { name: 'Monitors' }).click();
  const row = page.locator('.row', { hasText: name });
  await row.getByRole('button', { name: 'pause' }).click();
  await expect(row.getByText('PAUSED')).toBeVisible({ timeout: 5000 });
});
