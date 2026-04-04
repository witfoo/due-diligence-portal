/**
 * Global setup for Playwright E2E tests.
 * Logs in as admin and saves storageState for reuse across tests.
 */
import { chromium, type FullConfig } from '@playwright/test';
import path from 'path';
import fs from 'fs';

const AUTH_DIR = path.join(import.meta.dirname, '.auth');
const ADMIN_STATE = path.join(AUTH_DIR, 'admin.json');

async function globalSetup(config: FullConfig) {
	const baseURL = config.projects[0]?.use?.baseURL || 'http://localhost:8080';

	// Ensure auth directory exists.
	if (!fs.existsSync(AUTH_DIR)) {
		fs.mkdirSync(AUTH_DIR, { recursive: true });
	}

	// Login via API and save storageState.
	const browser = await chromium.launch();
	const context = await browser.newContext({ ignoreHTTPSErrors: true });
	const page = await context.newPage();

	// Login via API.
	const loginResp = await page.request.post(`${baseURL}/api/v1/auth/login`, {
		data: {
			email: process.env.DD_TEST_ADMIN_EMAIL || 'admin@localhost',
			password: process.env.DD_TEST_ADMIN_PASSWORD || 'testpass123'
		}
	});

	if (!loginResp.ok()) {
		throw new Error(`Login failed: ${loginResp.status()} ${await loginResp.text()}`);
	}

	const body = await loginResp.json();
	const token = body.data?.access_token;

	if (!token) {
		throw new Error('No access token in login response');
	}

	// Store token in localStorage by navigating to the app first.
	await page.goto(baseURL);
	await page.evaluate((t) => {
		localStorage.setItem('dd_auth_token', t);
	}, token);

	// Save storage state.
	await context.storageState({ path: ADMIN_STATE });
	await browser.close();

	console.log(`[E2E Setup] Admin auth saved to ${ADMIN_STATE}`);
}

export default globalSetup;
