import { test, expect } from '@playwright/test';

test.describe('Analytics Dashboard', () => {
	test('analytics page loads with heading', async ({ page }) => {
		await page.goto('/analytics', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('h1', { timeout: 10000 });
		await expect(page.locator('h1')).toContainText('Analytics Dashboard');
	});
});
