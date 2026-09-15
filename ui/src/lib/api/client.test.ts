import { describe, it, expect } from 'vitest';
import { ApiError, apiErrorMessage } from './client';

describe('apiErrorMessage', () => {
	it('returns the server error message from the envelope', () => {
		const err = new ApiError(400, 'Bad Request', {
			success: false,
			error: 'current password is incorrect'
		});
		expect(apiErrorMessage(err, 'fallback')).toBe('current password is incorrect');
	});

	it('falls back when the body is null', () => {
		expect(apiErrorMessage(new ApiError(500, 'Internal Server Error', null), 'fallback')).toBe(
			'fallback'
		);
	});

	it('falls back when the error field is empty', () => {
		const err = new ApiError(400, 'Bad Request', { success: false, error: '   ' });
		expect(apiErrorMessage(err, 'fallback')).toBe('fallback');
	});

	it('falls back when the error field is not a string', () => {
		const err = new ApiError(400, 'Bad Request', { success: false, error: { code: 1 } });
		expect(apiErrorMessage(err, 'fallback')).toBe('fallback');
	});

	it('falls back for non-API errors', () => {
		expect(apiErrorMessage(new TypeError('Failed to fetch'), 'fallback')).toBe('fallback');
		expect(apiErrorMessage(undefined, 'fallback')).toBe('fallback');
	});
});
