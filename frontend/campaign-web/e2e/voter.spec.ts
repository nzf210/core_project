import { test, expect } from '@playwright/test';

test.describe('Voter Registration', () => {
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

  test('should display voter registration form', async ({ page }) => {
    await page.goto('/voter');

    // Check for voter form fields
    await expect(page.locator('input[placeholder*="NIK"], input[id*="nik"]')).toBeVisible();
    await expect(page.locator('input[placeholder*="Nama"], input[id*="name"]')).toBeVisible();
  });

  test('should validate required fields', async ({ page }) => {
    await page.goto('/voter');

    // Try to submit without filling required fields
    const submitButton = page.locator('button[type="submit"], button:has-text("Registrasi")');
    await submitButton.click();

    // Form validation should prevent submission
    // Browser's built-in validation will trigger for required fields
    const nikInput = page.locator('input[placeholder*="NIK"], input[id*="nik"]');
    await expect(nikInput).toHaveAttribute('required', '');
  });
});
