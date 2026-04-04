import { test, expect } from '@playwright/test';

test.describe('Admin Pages', () => {
	test('admin branding page loads', async ({ page }) => {
		await page.goto('/admin/branding', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('h1', { timeout: 10000 });
		await expect(page.locator('h1')).toContainText('Branding');
	});

	test('admin watermark page loads', async ({ page }) => {
		await page.goto('/admin/watermark', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('h1', { timeout: 10000 });
		await expect(page.locator('h1')).toContainText('Watermark');
	});

	test('admin audit log page loads', async ({ page }) => {
		await page.goto('/admin/audit', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('h1', { timeout: 10000 });
		await expect(page.locator('h1')).toContainText('Audit Log');
	});

	test('admin navigation has all links', async ({ page }) => {
		await page.goto('/admin/branding', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('.admin-nav', { timeout: 10000 });
		const nav = page.locator('.admin-nav');
		await expect(nav.locator('a[href="/admin/users"]')).toBeVisible();
		await expect(nav.locator('a[href="/admin/categories"]')).toBeVisible();
		await expect(nav.locator('a[href="/admin/branding"]')).toBeVisible();
		await expect(nav.locator('a[href="/admin/watermark"]')).toBeVisible();
		await expect(nav.locator('a[href="/admin/nda"]')).toBeVisible();
		await expect(nav.locator('a[href="/admin/audit"]')).toBeVisible();
	});
});
