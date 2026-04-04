/**
 * Standalone Playwright script to seed data and capture documentation screenshots.
 *
 * Usage:
 *   cd ui && ./node_modules/.bin/tsx scripts-screenshots.ts
 *
 * Requires: running portal at http://localhost:9192 with admin/testpass123
 */
import { chromium } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';

const thisDir = path.dirname(fileURLToPath(import.meta.url));
const outDir = path.join(thisDir, '..', 'docs', 'screenshots');
const BASE = process.env.BASE_URL || 'http://localhost:9192';

async function main() {
	console.log(`Screenshots -> ${outDir}`);
	const browser = await chromium.launch({ headless: true });

	// Login.
	const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
	const tmp = await ctx.newPage();
	const lr = await tmp.request.post(`${BASE}/api/v1/auth/login`, {
		data: { email: 'admin@localhost', password: 'testpass123' }
	});
	const token = (await lr.json()).data.access_token;
	const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };

	// Seed documents.
	const docs = [
		['Series A Term Sheet', 'Draft term sheet for Series A round', 'cat-fundraising'],
		['Q3 2025 Financial Statements', 'Audited quarterly financials', 'cat-financials'],
		['Certificate of Incorporation', 'Delaware C-Corp certificate', 'cat-corporate'],
		['Patent Portfolio Summary', 'Overview of 12 filed patents', 'cat-ip'],
		['Product Roadmap 2026', 'Engineering roadmap and milestones', 'cat-product'],
		['Cap Table - Current', 'Current capitalization table', 'cat-financials'],
		['Employee Stock Option Plan', 'ESOP details and vesting', 'cat-team'],
		['SOC 2 Type II Report', 'Annual compliance report', 'cat-compliance'],
	];
	for (const [name, desc, cat] of docs) {
		await tmp.request.post(`${BASE}/api/v1/documents`, {
			headers: { Authorization: `Bearer ${token}` },
			multipart: {
				file: { name: name + '.pdf', mimeType: 'application/pdf', buffer: Buffer.from('Content: ' + name) },
				name, description: desc, category_id: cat
			}
		});
	}
	console.log(`Seeded ${docs.length} documents`);

	// Seed Q&A.
	for (const s of ['What is the current runway?', 'Can you share customer references?', 'What is the IP strategy?']) {
		await tmp.request.post(`${BASE}/api/v1/qa`, { headers, data: { subject: s } });
	}
	// Seed NDA.
	await tmp.request.post(`${BASE}/api/v1/nda/templates`, { headers, data: { name: 'Standard NDA', content: 'NDA content...' } });
	// Update branding.
	await tmp.request.put(`${BASE}/api/v1/branding`, { headers, data: { company_name: 'Acme Technologies' } });
	// View events.
	const dr = await tmp.request.get(`${BASE}/api/v1/documents`, { headers });
	for (const d of ((await dr.json()).data || []).slice(0, 5)) {
		await tmp.request.post(`${BASE}/api/v1/analytics/view-event`, { headers, data: { document_id: d.id, duration_ms: 30000, page_count: 5 } });
	}
	// Invite.
	await tmp.request.post(`${BASE}/api/v1/users/invite`, { headers, data: { email: 'investor@sequoia.com', role: 'investor' } });
	await ctx.close();
	console.log('Data seeded');

	const vp = { width: 1280, height: 800 };

	// Screenshot: Login.
	const c1 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: vp });
	const p1 = await c1.newPage();
	await p1.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded' });
	await p1.waitForSelector('h1', { timeout: 10000 });
	await p1.fill('#email', 'admin@acme.com');
	await p1.waitForTimeout(500);
	await p1.screenshot({ path: path.join(outDir, '01-login.png') });
	console.log('  01-login.png');
	await c1.close();

	// Authenticated screenshots.
	const c2 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: vp });
	const p = await c2.newPage();
	await p.goto(BASE, { waitUntil: 'domcontentloaded' });
	await p.evaluate((t: string) => localStorage.setItem('dd_auth_token', t), token);

	const pages: [string, string][] = [
		['/', '00-landing.png'],
		['/documents', '02-documents.png'],
		['/qa', '03-qa.png'],
		['/analytics', '04-analytics.png'],
		['/admin/branding', '05-admin-branding.png'],
		['/admin/watermark', '06-admin-watermark.png'],
		['/admin/audit', '07-admin-audit.png'],
	];
	for (const [route, file] of pages) {
		await p.goto(`${BASE}${route}`, { waitUntil: 'domcontentloaded' });
		await p.waitForSelector('h1', { timeout: 10000 });
		await p.waitForTimeout(2000);
		await p.screenshot({ path: path.join(outDir, file) });
		console.log(`  ${file}`);
	}
	await c2.close();
	await browser.close();
	console.log('Done!');
}

main().catch(e => { console.error(e); process.exit(1); });
