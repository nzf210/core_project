# End-to-End Testing Guide — Playwright

**Date:** 2026-08-17  
**Scope:** P2-8 — E2E test setup for frontend applications  
**Framework:** Playwright  
**Coverage:** UMKM Web, Campaign Web

---

## Overview

E2E (End-to-End) tests validate complete user workflows from the browser perspective. Unlike unit tests (test individual functions) and component tests (test Vue components in isolation), E2E tests:

- Run in a real browser (Chromium)
- Test the full application stack (frontend + backend integration)
- Verify user-facing behavior and critical paths
- Catch integration issues that unit tests miss

**Test pyramid:**
```
    E2E Tests (few, slow, high confidence)
        ↑
   Component Tests (more, faster)
        ↑
  Unit Tests (many, fast, focused)
```

---

## Setup Complete

### Installed Packages

**UMKM Web:**
```json
{
  "devDependencies": {
    "@playwright/test": "^1.62.1"
  }
}
```

**Campaign Web:**
```json
{
  "devDependencies": {
    "@playwright/test": "^1.62.1"
  }
}
```

### Project Structure

```
frontend/umkm-web/
├── playwright.config.ts        # Playwright configuration
├── e2e/                        # E2E test files
│   ├── auth.spec.ts           # Authentication flow tests
│   ├── dashboard.spec.ts      # Dashboard navigation tests
│   ├── wa-setup.spec.ts       # WhatsApp setup tests
│   └── chatbot.spec.ts        # Chatbot configuration tests
└── package.json               # Added E2E scripts

frontend/campaign-web/
├── playwright.config.ts        # Playwright configuration
├── e2e/                        # E2E test files
│   ├── dashboard.spec.ts      # Campaign dashboard tests
│   └── voter.spec.ts          # Voter registration tests
└── package.json               # Added E2E scripts
```

---

## Running E2E Tests

### UMKM Web

```bash
cd frontend/umkm-web

# Run all E2E tests (headless mode)
npm run test:e2e

# Run with UI mode (recommended for development)
npm run test:e2e:ui

# Run in headed mode (see browser)
npm run test:e2e:headed

# Debug mode (step through tests)
npm run test:e2e:debug

# Run specific test file
npx playwright test e2e/auth.spec.ts

# Run tests matching pattern
npx playwright test --grep "login"
```

### Campaign Web

```bash
cd frontend/campaign-web

# Same commands as UMKM Web
npm run test:e2e
npm run test:e2e:ui
npm run test:e2e:headed
npm run test:e2e:debug
```

---

## Configuration

### UMKM Web (`playwright.config.ts`)

```typescript
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',

  use: {
    baseURL: 'http://localhost:3201',  // Vite dev server
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3201',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
});
```

**Key settings:**
- `baseURL`: Base URL for `page.goto('/')` calls
- `webServer`: Auto-starts dev server before tests
- `reuseExistingServer`: Reuses running dev server in local development
- `retries`: Retry flaky tests in CI (2x), no retry locally
- `workers`: Single worker in CI (more stable), parallel locally

### Campaign Web (`playwright.config.ts`)

Same structure, different port:
- `baseURL`: `http://localhost:3301`
- `webServer.url`: `http://localhost:3301`

---

## Test Examples

### Authentication Flow (`umkm-web/e2e/auth.spec.ts`)

```typescript
test('should display login page', async ({ page }) => {
  await page.goto('/');

  // Check if login form is visible
  await expect(page.locator('input[type="text"]')).toBeVisible();
  await expect(page.locator('button:has-text("Login")')).toBeVisible();
});

test('should navigate to OTP input after phone submission', async ({ page }) => {
  await page.goto('/');

  // Enter valid phone number
  await page.locator('input[type="text"]').fill('628123456789');
  await page.locator('button:has-text("Login")').click();

  // Should show OTP input
  await expect(page.locator('text=/OTP|kode/i')).toBeVisible({ timeout: 10000 });
});
```

### Dashboard Navigation (`umkm-web/e2e/dashboard.spec.ts`)

```typescript
test.beforeEach(async ({ page }) => {
  // Mock authentication state
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

test('should navigate to settings page', async ({ page }) => {
  await page.goto('/dashboard');

  await page.locator('a:has-text("Settings")').first().click();

  await expect(page).toHaveURL(/\/settings/);
});
```

### Voter Registration (`campaign-web/e2e/voter.spec.ts`)

```typescript
test('should display voter registration form', async ({ page }) => {
  await page.goto('/voter');

  // Check for voter form fields
  await expect(page.locator('input[placeholder*="NIK"]')).toBeVisible();
  await expect(page.locator('input[placeholder*="Nama"]')).toBeVisible();
});
```

---

## Writing E2E Tests

### Test Structure

```typescript
import { test, expect } from '@playwright/test';

test.describe('Feature Name', () => {
  test.beforeEach(async ({ page }) => {
    // Setup: navigate, mock auth, etc.
  });

  test('should do something', async ({ page }) => {
    // 1. Arrange: navigate, set state
    await page.goto('/some-page');

    // 2. Act: interact with UI
    await page.locator('button').click();

    // 3. Assert: verify outcome
    await expect(page.locator('text=Success')).toBeVisible();
  });

  test.afterEach(async ({ page }) => {
    // Cleanup if needed
  });
});
```

### Locator Best Practices

**Priority order (most stable to least stable):**

1. **Role-based** (accessible, semantic):
   ```typescript
   page.getByRole('button', { name: 'Submit' })
   page.getByRole('textbox', { name: 'Email' })
   ```

2. **Label/Placeholder** (user-facing text):
   ```typescript
   page.getByLabel('Email address')
   page.getByPlaceholder('Enter your email')
   ```

3. **Test ID** (explicit test hooks):
   ```typescript
   page.getByTestId('submit-button')
   // Requires: <button data-testid="submit-button">
   ```

4. **Text content** (visible to user):
   ```typescript
   page.locator('text=Login')
   page.locator('button:has-text("Submit")')
   ```

5. **CSS selectors** (last resort, brittle):
   ```typescript
   page.locator('.btn-primary')
   page.locator('#submit-btn')
   ```

**Avoid:**
- XPath selectors (hard to read, brittle)
- Deep CSS nesting (`.container .sidebar .menu .item`)

### Mocking Authentication

Most tests need authenticated state. Use `beforeEach` to mock:

```typescript
test.beforeEach(async ({ page }) => {
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
```

**Alternative:** Use Playwright's `storageState` to save/load auth once:

```typescript
// auth.setup.ts
test('authenticate', async ({ page }) => {
  await page.goto('/login');
  await page.fill('input[type="text"]', '628123456789');
  await page.click('button:has-text("Login")');
  await page.fill('input[type="text"]', '123456'); // OTP
  await page.click('button:has-text("Verify")');
  
  // Save auth state
  await page.context().storageState({ path: 'auth.json' });
});

// Other tests
test.use({ storageState: 'auth.json' });
```

### Waiting for Elements

**Auto-waiting (preferred):**
```typescript
// Playwright automatically waits for element to be visible & enabled
await page.locator('button').click();
await expect(page.locator('text=Success')).toBeVisible();
```

**Explicit timeout:**
```typescript
await expect(page.locator('text=Loaded')).toBeVisible({ timeout: 10000 });
```

**Wait for navigation:**
```typescript
await Promise.all([
  page.waitForNavigation(),
  page.click('a[href="/dashboard"]')
]);
```

**Wait for API response:**
```typescript
await page.waitForResponse(resp => 
  resp.url().includes('/api/users') && resp.status() === 200
);
```

### Assertions

```typescript
// Visibility
await expect(page.locator('button')).toBeVisible();
await expect(page.locator('.error')).toBeHidden();

// Text content
await expect(page.locator('h1')).toHaveText('Dashboard');
await expect(page.locator('.message')).toContainText('Success');

// URL
await expect(page).toHaveURL('/dashboard');
await expect(page).toHaveURL(/\/settings/);

// Attributes
await expect(page.locator('input')).toHaveAttribute('required');
await expect(page.locator('button')).toBeEnabled();
await expect(page.locator('input[type="checkbox"]')).toBeChecked();

// Count
await expect(page.locator('.item')).toHaveCount(5);
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '20'
      
      - name: Install dependencies
        run: |
          cd frontend/umkm-web
          npm ci
      
      - name: Install Playwright browsers
        run: npx playwright install --with-deps chromium
      
      - name: Run E2E tests
        run: npm run test:e2e
      
      - name: Upload test report
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: playwright-report
          path: playwright-report/
```

### Environment Variables

**Development:**
```bash
# .env.local (frontend)
VITE_API_URL=http://localhost:8000
```

**CI:**
```yaml
env:
  CI: true
  VITE_API_URL: http://localhost:8000
```

---

## Test Coverage

### UMKM Web Tests

| Test File | Coverage | Critical Paths |
|:----------|:---------|:---------------|
| `auth.spec.ts` | Authentication | Login form, OTP flow, validation errors |
| `dashboard.spec.ts` | Navigation | Dashboard display, menu navigation, page routing |
| `wa-setup.spec.ts` | WhatsApp Setup | QR code display, connection status |
| `chatbot.spec.ts` | Chatbot Config | Config page, toggle activation |

**Total:** 11 test cases covering 4 user flows

### Campaign Web Tests

| Test File | Coverage | Critical Paths |
|:----------|:---------|:---------------|
| `dashboard.spec.ts` | Navigation | Dashboard display, volunteer/real count navigation |
| `voter.spec.ts` | Voter Registration | Form display, field validation |

**Total:** 5 test cases covering 2 user flows

---

## Common Issues & Solutions

### Issue 1: Dev server not starting

**Error:**
```
Error: webServer command "npm run dev" exited with code 1
```

**Solution:**
```bash
# Check if port is already in use
lsof -ti:3201 | xargs kill -9

# Or change port in vite.config.ts and playwright.config.ts
```

### Issue 2: Test timeout

**Error:**
```
Timeout 30000ms exceeded while waiting for element
```

**Solution:**
```typescript
// Increase timeout for slow elements
await expect(page.locator('text=Data loaded')).toBeVisible({ timeout: 10000 });

// Or globally in playwright.config.ts
expect: {
  timeout: 10000
}
```

### Issue 3: Element not clickable

**Error:**
```
Element is not clickable at point (x, y)
```

**Solution:**
```typescript
// Wait for element to be ready
await page.locator('button').waitFor({ state: 'visible' });
await page.locator('button').click();

// Or scroll into view first
await page.locator('button').scrollIntoViewIfNeeded();
await page.locator('button').click();
```

### Issue 4: Flaky tests

**Symptoms:** Tests pass locally but fail in CI, or pass/fail intermittently.

**Solutions:**
1. **Add explicit waits:**
   ```typescript
   await page.waitForLoadState('networkidle');
   ```

2. **Use retries in CI:**
   ```typescript
   retries: process.env.CI ? 2 : 0
   ```

3. **Avoid hard-coded delays:**
   ```typescript
   // ❌ Bad
   await page.waitForTimeout(5000);
   
   // ✅ Good
   await expect(page.locator('text=Loaded')).toBeVisible();
   ```

---

## Debugging Tests

### 1. UI Mode (Recommended)

```bash
npm run test:e2e:ui
```

- **Time travel:** Step through test execution
- **Watch mode:** Auto-rerun on file changes
- **Locator picker:** Click elements to generate selectors

### 2. Debug Mode

```bash
npm run test:e2e:debug
```

- Opens Playwright Inspector
- Step through test line-by-line
- Inspect page state at each step

### 3. Headed Mode

```bash
npm run test:e2e:headed
```

- See browser window during test execution
- Useful for visual debugging

### 4. Screenshots & Traces

**Automatic on failure:**
```typescript
// playwright.config.ts
use: {
  screenshot: 'only-on-failure',
  trace: 'on-first-retry',
}
```

**Manual screenshots:**
```typescript
await page.screenshot({ path: 'debug.png' });
await page.locator('.section').screenshot({ path: 'section.png' });
```

**View trace:**
```bash
npx playwright show-trace trace.zip
```

---

## Best Practices

### 1. Test User Journeys, Not Implementation

**❌ Bad (tests implementation):**
```typescript
test('should call API when button clicked', async ({ page }) => {
  await page.route('/api/save', route => route.fulfill({ body: '{}' }));
  await page.click('button');
  // Brittle - tests internal API call
});
```

**✅ Good (tests user outcome):**
```typescript
test('should show success message after save', async ({ page }) => {
  await page.click('button:has-text("Save")');
  await expect(page.locator('text=Saved successfully')).toBeVisible();
  // Tests what user sees
});
```

### 2. Keep Tests Independent

Each test should:
- Set up its own state
- Clean up after itself
- Not depend on other tests

```typescript
test.beforeEach(async ({ page }) => {
  // Reset state for each test
  await page.goto('/');
  await page.evaluate(() => localStorage.clear());
});
```

### 3. Use Page Object Model for Complex Pages

```typescript
// pages/LoginPage.ts
export class LoginPage {
  constructor(private page: Page) {}

  async goto() {
    await this.page.goto('/login');
  }

  async login(phone: string, otp: string) {
    await this.page.fill('input[type="text"]', phone);
    await this.page.click('button:has-text("Login")');
    await this.page.fill('input[type="text"]', otp);
    await this.page.click('button:has-text("Verify")');
  }

  async expectDashboard() {
    await expect(this.page).toHaveURL('/dashboard');
  }
}

// auth.spec.ts
test('login flow', async ({ page }) => {
  const loginPage = new LoginPage(page);
  await loginPage.goto();
  await loginPage.login('628123456789', '123456');
  await loginPage.expectDashboard();
});
```

### 4. Test Critical Paths Only

E2E tests are expensive (slow, flaky). Focus on:
- Happy path for critical flows (login, checkout, form submission)
- Edge cases that broke production before
- Cross-browser compatibility issues

Don't test:
- Every button color
- Every validation message
- Implementation details (use unit tests)

---

## Next Steps

### Immediate (Post-Setup)

1. **Run existing tests:**
   ```bash
   cd frontend/umkm-web && npm run test:e2e:ui
   cd frontend/campaign-web && npm run test:e2e:ui
   ```

2. **Fix any failing tests** due to UI changes since tests were written

3. **Add CI integration** (GitHub Actions workflow)

### Short-Term (Next Sprint)

1. **Expand coverage:**
   - Transaction creation flow (UMKM)
   - Product catalog CRUD (UMKM)
   - Volunteer registration (Campaign)
   - Real count data entry (Campaign)

2. **Add API mocking** for tests that don't need real backend:
   ```typescript
   await page.route('/api/**', route => {
     route.fulfill({ body: JSON.stringify({ success: true }) });
   });
   ```

3. **Setup visual regression testing** (screenshot comparison):
   ```bash
   npm install -D @playwright/test
   # Use toHaveScreenshot() assertions
   ```

### Long-Term

1. **Cross-browser testing** (Firefox, Safari/WebKit)
2. **Mobile viewport testing** (responsive design)
3. **Performance testing** (Lighthouse integration)
4. **Accessibility testing** (axe-core integration)

---

## Resources

- **Playwright Docs:** https://playwright.dev/docs/intro
- **Best Practices:** https://playwright.dev/docs/best-practices
- **API Reference:** https://playwright.dev/docs/api/class-playwright
- **Examples:** https://github.com/microsoft/playwright/tree/main/examples

---

## Summary

**Installed:** Playwright E2E testing framework in `umkm-web` and `campaign-web`

**Created:**
- Configuration files (`playwright.config.ts`)
- 11 E2E tests for UMKM Web (auth, dashboard, WA setup, chatbot)
- 5 E2E tests for Campaign Web (dashboard, voter registration)
- npm scripts for running tests (headless, UI mode, headed, debug)

**Commands:**
```bash
npm run test:e2e          # Run tests headless
npm run test:e2e:ui       # UI mode (recommended)
npm run test:e2e:headed   # See browser
npm run test:e2e:debug    # Debug mode
```

**Coverage:** Critical user flows for authentication, navigation, and core features.

**Next:** Expand test coverage, add CI integration, fix any failing tests.
