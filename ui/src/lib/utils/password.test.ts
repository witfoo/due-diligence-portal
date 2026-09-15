import { describe, it, expect } from 'vitest';
import { PASSWORD_MAX_BYTES, PASSWORD_MIN_LENGTH, validateNewPassword } from './password';

describe('password constants', () => {
	it('match the server-side limits', () => {
		expect(PASSWORD_MIN_LENGTH).toBe(8);
		expect(PASSWORD_MAX_BYTES).toBe(72);
	});
});

describe('validateNewPassword', () => {
	it('requires a password', () => {
		expect(validateNewPassword('', '')).toBe('Password is required.');
	});

	it('rejects passwords shorter than 8 characters', () => {
		expect(validateNewPassword('short12', 'short12')).toBe(
			'Password must be at least 8 characters.'
		);
	});

	it('accepts a password of exactly 8 characters', () => {
		expect(validateNewPassword('exactly8', 'exactly8')).toBeNull();
	});

	it('counts multi-byte characters once toward the minimum length', () => {
		// 4 emoji = 8 UTF-16 code units and 16 bytes, but only 4 characters.
		const pw = '\u{1F600}\u{1F600}\u{1F600}\u{1F600}';
		expect(validateNewPassword(pw, pw)).toBe('Password must be at least 8 characters.');
	});

	it('accepts a password of exactly 72 bytes', () => {
		const pw = 'a'.repeat(72);
		expect(validateNewPassword(pw, pw)).toBeNull();
	});

	it('rejects a password longer than 72 bytes', () => {
		const pw = 'a'.repeat(73);
		expect(validateNewPassword(pw, pw)).toContain('at most 72 bytes');
	});

	it('accepts multi-byte characters landing exactly on the byte limit', () => {
		// 'é' is 2 bytes in UTF-8: 36 characters = 72 bytes.
		const pw = 'é'.repeat(36);
		expect(new TextEncoder().encode(pw).length).toBe(72);
		expect(validateNewPassword(pw, pw)).toBeNull();
	});

	it('rejects multi-byte characters crossing the byte limit', () => {
		// 37 characters but 74 bytes: under 72 characters, over 72 bytes.
		const pw = 'é'.repeat(37);
		expect(validateNewPassword(pw, pw)).toContain('at most 72 bytes');
	});

	it('rejects a 4-byte character that pushes past the byte limit', () => {
		// 69 ASCII bytes + one 4-byte emoji = 73 bytes.
		const pw = 'a'.repeat(69) + '\u{1F600}';
		expect(validateNewPassword(pw, pw)).toContain('at most 72 bytes');
	});

	it('rejects mismatched confirmation', () => {
		expect(validateNewPassword('correct-horse', 'correct-horsE')).toBe('Passwords do not match.');
	});

	it('rejects an empty confirmation', () => {
		expect(validateNewPassword('correct-horse', '')).toBe('Passwords do not match.');
	});

	it('returns null for a valid, matching password', () => {
		expect(validateNewPassword('correct-horse-battery', 'correct-horse-battery')).toBeNull();
	});
});
