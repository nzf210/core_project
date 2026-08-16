import { test, expect } from '@playwright/test';

test.describe('Campaign Dashboard', () => {
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

  test('should display campaign dashboard', async ({ page }) => {
    await page.goto('/dashboard');

    // Check for campaign dashboard elements
    await expect(page.locator('text=/campaign|kampanye/i')).toBeVisible();
  });

  test('should navigate to volunteers page', async ({ page }) => {
    await page.goto('/dashboard');

    // Click volunteers link
    await page.locator('a:has-text("Volunteer"), a:has-text("Relawan")').first().click();

    // Should be on volunteers page
    await expect(page).toHaveURL(/\/volunteer/);
  });

  test('should navigate to real count page', async ({ page }) => {
    await page.goto('/dashboard');

    // Click real count link
    await page.locator('a:has-text("Real Count"), a:has-text("Hasil")').first().click();

    // Should be on real count page
    await expect(page).toHaveURL(/\/realcount/);
    await expect(page.locator('text=/real count|hasil/i')).toBeVisible();
  });
});
