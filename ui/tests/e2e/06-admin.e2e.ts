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

	// Non-mutating: opens and cancels the form without submitting, so the e2e
	// admin credentials never change.
	test('admin users page opens and cancels the set password form', async ({ page }) => {
		await page.goto('/admin/users', { waitUntil: 'domcontentloaded' });
		await page.waitForSelector('table tbody tr', { timeout: 10000 });

		await page.getByRole('button', { name: 'Set Password', exact: true }).first().click();

		const newPassword = page.locator('.password-form input[name="new_password"]');
		const confirmPassword = page.locator('.password-form input[name="confirm_password"]');
		await expect(newPassword).toBeVisible();
		await expect(confirmPassword).toBeVisible();

		await page.locator('.password-form').getByRole('button', { name: 'Cancel', exact: true }).click();

		await expect(newPassword).toHaveCount(0);
		await expect(confirmPassword).toHaveCount(0);
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
