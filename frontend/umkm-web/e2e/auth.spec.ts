import { test, expect } from '@playwright/test';

test.describe('Authentication Flow', () => {
  test('should display login page', async ({ page }) => {
    await page.goto('/');

    // Check if login form is visible
    await expect(page.locator('input[type="text"]')).toBeVisible();
    await expect(page.locator('button:has-text("Login")')).toBeVisible();
  });

  test('should show validation error for empty phone', async ({ page }) => {
    await page.goto('/');

    // Try to login without phone number
    await page.locator('button:has-text("Login")').click();

    // Should show validation message
    await expect(page.locator('text=/phone.*required/i')).toBeVisible();
  });

  test('should navigate to OTP input after phone submission', async ({ page }) => {
    await page.goto('/');

    // Enter valid phone number
    await page.locator('input[type="text"]').fill('628123456789');
    await page.locator('button:has-text("Login")').click();

    // Should show OTP input
    await expect(page.locator('text=/OTP|kode/i')).toBeVisible({ timeout: 10000 });
  });
});
