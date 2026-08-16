import { test, expect } from '@playwright/test';

test.describe('WhatsApp Setup Flow', () => {
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

  test('should display WA setup page', async ({ page }) => {
    await page.goto('/settings');

    // Look for WA setup section
    await expect(page.locator('text=/whatsapp|wa setup|scan qr/i')).toBeVisible();
  });

  test('should show QR code button when not connected', async ({ page }) => {
    await page.goto('/settings');

    // Should have button to generate QR
    const qrButton = page.locator('button:has-text("QR"), button:has-text("Scan")');
    await expect(qrButton.first()).toBeVisible();
  });
});
