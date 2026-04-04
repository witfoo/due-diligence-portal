import { test, expect } from '@playwright/test';

test.describe('Q&A Page', () => {
	test('Q&A page loads with heading', async ({ page }) => {
		await page.goto('/qa', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('h1', { timeout: 10000 });
		await expect(page.locator('h1')).toContainText('Questions & Answers');
	});

	test('Q&A page has ask question button', async ({ page }) => {
		await page.goto('/qa', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('button', { timeout: 10000 });
		const askBtn = page.locator('button', { hasText: 'Ask Question' });
		await expect(askBtn).toBeVisible();
	});

	test('clicking Ask Question shows input form', async ({ page }) => {
		await page.goto('/qa', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('button', { timeout: 10000 });
		await page.locator('button', { hasText: 'Ask Question' }).click();
		const input = page.locator('input[placeholder*="What would you like"]');
		await expect(input).toBeVisible();
	});
});
