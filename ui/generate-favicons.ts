/**
 * Generate favicon PNGs from SVG using Playwright.
 * Run: cd ui && ./node_modules/.bin/tsx generate-favicons.ts
 */
import { chromium } from '@playwright/test';
import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const staticDir = path.join(__dirname, 'static');
const svgPath = path.join(staticDir, 'favicon.svg');

const sizes = [
	{ name: 'favicon-16x16.png', size: 16 },
	{ name: 'favicon-32x32.png', size: 32 },
	{ name: 'favicon.png', size: 128 },
	{ name: 'apple-touch-icon.png', size: 180 },
	{ name: 'icon-192.png', size: 192 },
	{ name: 'icon-512.png', size: 512 },
];

async function main() {
	const svgContent = fs.readFileSync(svgPath, 'utf-8');
	const browser = await chromium.launch({ headless: true });

	for (const { name, size } of sizes) {
		const ctx = await browser.newContext({ viewport: { width: size, height: size } });
		const page = await ctx.newPage();
		await page.setContent(`
			<html><body style="margin:0;padding:0;background:transparent">
				<div style="width:${size}px;height:${size}px">${svgContent}</div>
			</body></html>
		`);
		await page.screenshot({
			path: path.join(staticDir, name),
			omitBackground: true,
			clip: { x: 0, y: 0, width: size, height: size }
		});
		await ctx.close();
		console.log(`  ${name} (${size}x${size})`);
	}

	await browser.close();

	// Generate ICO from the 16x16 and 32x32 PNGs (ICO is just a container).
	// For maximum compat, we'll reference the PNGs directly instead.
	console.log('\nGenerated all favicon PNGs.');
	console.log('SVG favicon at static/favicon.svg (modern browsers)');
}

main().catch(e => { console.error(e); process.exit(1); });
