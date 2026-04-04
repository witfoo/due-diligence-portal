import { describe, it, expect } from 'vitest';
import { formatFileSize, formatDate, formatDateTime, truncate } from './format';

describe('formatFileSize', () => {
	it('formats bytes', () => {
		expect(formatFileSize(500)).toBe('500 B');
	});

	it('formats kilobytes', () => {
		expect(formatFileSize(1536)).toBe('1.5 KB');
	});

	it('formats megabytes', () => {
		expect(formatFileSize(5 * 1024 * 1024)).toBe('5.0 MB');
	});

	it('formats gigabytes', () => {
		expect(formatFileSize(2.5 * 1024 * 1024 * 1024)).toBe('2.5 GB');
	});

	it('handles zero', () => {
		expect(formatFileSize(0)).toBe('0 B');
	});
});

describe('formatDate', () => {
	it('formats ISO date string', () => {
		const result = formatDate('2026-04-03T10:30:00Z');
		expect(result).toContain('2026');
		expect(result).toContain('Apr');
	});
});

describe('formatDateTime', () => {
	it('formats ISO date+time string', () => {
		const result = formatDateTime('2026-04-03T10:30:00Z');
		expect(result).toContain('Apr');
	});
});

describe('truncate', () => {
	it('returns short strings unchanged', () => {
		expect(truncate('hello', 10)).toBe('hello');
	});

	it('truncates long strings with ellipsis', () => {
		expect(truncate('hello world', 8)).toBe('hello w\u2026');
	});

	it('handles exact length', () => {
		expect(truncate('hello', 5)).toBe('hello');
	});
});
