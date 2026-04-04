import { test, expect } from '@playwright/test';

test.describe('Documents Page', () => {
	test('documents page loads with heading', async ({ page }) => {
		await page.goto('/documents', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('h1', { timeout: 10000 });
		await expect(page.locator('h1')).toContainText('Documents');
	});

	test('documents page has search and category filter', async ({ page }) => {
		await page.goto('/documents', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('.filters', { timeout: 10000 });
		await expect(page.locator('input[placeholder*="Search"]')).toBeVisible();
		await expect(page.locator('select')).toBeVisible();
	});
});
