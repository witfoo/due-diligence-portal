import { test, expect } from '@playwright/test';

test.describe('Authentication Flow', () => {
	test('login page loads', async ({ page }) => {
		await page.goto('/login');
		await expect(page.locator('h1')).toContainText('Sign In');
	});

	test('login page has email and password fields', async ({ page }) => {
		await page.goto('/login');
		await expect(page.locator('#email')).toBeVisible();
		await expect(page.locator('#password')).toBeVisible();
	});

	test('login page has submit button', async ({ page }) => {
		await page.goto('/login');
		const submitBtn = page.locator('button[type="submit"]');
		await expect(submitBtn).toBeVisible();
		await expect(submitBtn).toContainText('Sign In');
	});

	test('health endpoint is accessible', async ({ request }) => {
		const resp = await request.get('/health');
		expect(resp.ok()).toBeTruthy();
		const body = await resp.json();
		expect(body.status).toBe('healthy');
	});

	test('ready endpoint is accessible', async ({ request }) => {
		const resp = await request.get('/ready');
		expect(resp.ok()).toBeTruthy();
		const body = await resp.json();
		expect(body.status).toBe('ready');
		expect(body.checks.sqlite).toBe('ok');
	});

	test('version endpoint is accessible', async ({ request }) => {
		const resp = await request.get('/version');
		expect(resp.ok()).toBeTruthy();
		const body = await resp.json();
		expect(body.version).toBeDefined();
	});

	test('API returns 401 without auth', async ({ request }) => {
		const resp = await request.get('/api/v1/auth/me');
		expect(resp.status()).toBe(401);
	});

	test('API login with wrong password returns 401', async ({ request }) => {
		const resp = await request.post('/api/v1/auth/login', {
			data: { email: 'admin@localhost', password: 'wrongpassword' }
		});
		expect(resp.status()).toBe(401);
	});
});
