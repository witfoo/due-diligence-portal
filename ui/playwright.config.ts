import { defineConfig, devices } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const ADMIN_STORAGE_STATE = path.join(__dirname, 'tests/e2e/.auth/admin.json');

export default defineConfig({
	testDir: './tests/e2e',
	testMatch: '**/*.e2e.ts',
	globalSetup: path.join(__dirname, 'tests/e2e/global-setup.ts'),
	timeout: 30_000,
	expect: { timeout: 15_000 },
	fullyParallel: true,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 2 : 1,
	workers: process.env.CI ? 1 : 4,
	reporter: process.env.PLAYWRIGHT_REPORTER === 'html'
		? [
				['html', { outputFolder: 'playwright-report' }],
				['json', { outputFile: 'playwright-report/results.json' }],
				['list']
			]
		: [['list']],
	use: {
		baseURL: process.env.BASE_URL || 'http://localhost:8080',
		ignoreHTTPSErrors: true,
		headless: process.env.HEADED !== '1',
		screenshot: 'only-on-failure',
		video: 'retain-on-failure',
		trace: 'on-first-retry',
		actionTimeout: 10_000,
		navigationTimeout: 15_000
	},
	projects: [
		{
			name: 'chromium',
			use: {
				...devices['Desktop Chrome'],
				storageState: ADMIN_STORAGE_STATE
			},
			testIgnore: ['**/01-auth.e2e.ts', '**/81-*.e2e.ts']
		},
		{
			name: 'chromium-no-auth',
			use: { ...devices['Desktop Chrome'] },
			testMatch: ['**/01-auth.e2e.ts']
		},
		{
			name: 'chromium-role-auth',
			use: { ...devices['Desktop Chrome'] },
			testMatch: ['**/81-*.e2e.ts']
		}
	]
});
