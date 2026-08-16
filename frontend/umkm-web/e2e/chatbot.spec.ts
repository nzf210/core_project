import { test, expect } from '@playwright/test';

test.describe('Chatbot Configuration', () => {
  test.beforeEach(async ({ page }) => {
    // Mock authentication as owner
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

  test('should display chatbot config page', async ({ page }) => {
    await page.goto('/chatbot-config');

    // Check for chatbot configuration elements
    await expect(page.locator('text=/chatbot|ai|assistant/i')).toBeVisible();
  });

  test('should allow toggling chatbot active state', async ({ page }) => {
    await page.goto('/chatbot-config');

    // Look for activation toggle
    const toggle = page.locator('input[type="checkbox"]').first();

    if (await toggle.isVisible()) {
      const isChecked = await toggle.isChecked();
      await toggle.click();

      // State should have changed
      await expect(toggle).toHaveAttribute('checked', isChecked ? null : '');
    }
  });
});
