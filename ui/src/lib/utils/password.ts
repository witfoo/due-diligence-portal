/**
 * Password validation shared by forms that set a new password.
 * Mirrors the server-side rules in domain.ValidatePassword.
 */

/** Minimum password length, in characters. */
export const PASSWORD_MIN_LENGTH = 8;

/** Maximum password length, in UTF-8 bytes (bcrypt ignores anything beyond 72 bytes). */
export const PASSWORD_MAX_BYTES = 72;

/**
 * Validate a new password and its confirmation.
 * Returns a user-facing error message, or null when the password is acceptable.
 * Length is counted in Unicode code points so multi-byte characters count once,
 * while the upper bound is measured in UTF-8 bytes to match bcrypt's limit.
 */
export function validateNewPassword(password: string, confirm: string): string | null {
	if (!password) return 'Password is required.';
	if (Array.from(password).length < PASSWORD_MIN_LENGTH) {
		return `Password must be at least ${PASSWORD_MIN_LENGTH} characters.`;
	}
	if (new TextEncoder().encode(password).length > PASSWORD_MAX_BYTES) {
		return `Password must be at most ${PASSWORD_MAX_BYTES} bytes. Characters outside basic Latin count as more than one byte.`;
	}
	// Compares the user's own two form inputs in the browser; there is no secret to leak via timing.
	// eslint-disable-next-line security/detect-possible-timing-attacks
	if (password !== confirm) return 'Passwords do not match.';
	return null;
}
