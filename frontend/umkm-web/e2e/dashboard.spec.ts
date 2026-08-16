import { test, expect } from '@playwright/test';

test.describe('Dashboard Navigation', () => {
  test.beforeEach(async ({ page }) => {
    // Mock authentication
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('token', 'mock-jwt-token');
      localStorage.setItem('user', JSON.stringify({
        id: 'test-user-id',
        role: 'owner',
        tenant_id: 'test-tenant-id'
      }));
    });
  });

  test('should display main dashboard after login', async ({ page }) => {
    await page.goto('/dashboard');

    // Check for dashboard elements
    await expect(page.locator('text=/dashboard|ringkasan/i')).toBeVisible();
  });

  test('should navigate to settings page', async ({ page }) => {
    await page.goto('/dashboard');

    // Click settings link
    await page.locator('a:has-text("Settings"), a:has-text("Pengaturan")').first().click();

    // Should be on settings page
    await expect(page).toHaveURL(/\/settings/);
    await expect(page.locator('text=/pengaturan|settings/i')).toBeVisible();
  });

  test('should navigate to transactions page', async ({ page }) => {
    await page.goto('/dashboard');

    // Click transactions/journal link
    await page.locator('a:has-text("Journal"), a:has-text("Transaksi")').first().click();

    // Should be on transactions page
    await expect(page).toHaveURL(/\/journal|\/transactions/);
  });
});
