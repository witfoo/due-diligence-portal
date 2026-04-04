import type { Page } from '@playwright/test';

/**
 * Wait for Svelte 5 reactivity to settle (double requestAnimationFrame).
 */
export async function waitForSvelteUpdate(page: Page): Promise<void> {
	await page.evaluate(
		() =>
			new Promise<void>((resolve) => {
				requestAnimationFrame(() => {
					requestAnimationFrame(() => resolve());
				});
			})
	);
}

/**
 * Wait for a table to have at least `minRows` rows loaded.
 */
export async function waitForTableLoad(
	page: Page,
	minRows = 1,
	timeout = 5000
): Promise<boolean> {
	try {
		await page.waitForFunction(
			(min) => document.querySelectorAll('[role="row"]').length >= min,
			minRows,
			{ timeout }
		);
		return true;
	} catch {
		return false;
	}
}

/**
 * Wait for a toast notification to appear.
 */
export async function waitForToast(page: Page, timeout = 5000): Promise<string | null> {
	try {
		const toast = page.locator('[role="alert"]').first();
		await toast.waitFor({ state: 'visible', timeout });
		return toast.textContent();
	} catch {
		return null;
	}
}

/**
 * Wait for input debounce to complete and UI to update.
 */
export async function waitForDebounceAndUpdate(page: Page, debounceMs = 350): Promise<void> {
	await page.waitForTimeout(debounceMs);
	await waitForSvelteUpdate(page);
}
