import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
	test('landing page loads with portal title', async ({ page }) => {
		await page.goto('/', { waitUntil: 'domcontentloaded' });
		// Wait for Svelte to hydrate.
		await page.waitForSelector('h1', { timeout: 10000 });
		await expect(page.locator('h1')).toContainText('Due Diligence Portal');
	});

	test('landing page has sign in link', async ({ page }) => {
		await page.goto('/', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('a', { timeout: 10000 });
		const signIn = page.locator('a[href="/login"]');
		await expect(signIn).toBeVisible();
	});
});
