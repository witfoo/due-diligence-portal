/**
 * Formatting utilities for the portal UI.
 */

/**
 * Format a byte count as a human-readable file size.
 */
export function formatFileSize(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

/**
 * Format an ISO date string as a short locale date.
 */
export function formatDate(iso: string): string {
	return new Date(iso).toLocaleDateString('en-US', {
		year: 'numeric',
		month: 'short',
		day: 'numeric'
	});
}

/**
 * Format an ISO date string as a date+time string.
 */
export function formatDateTime(iso: string): string {
	return new Date(iso).toLocaleString('en-US', {
		month: 'short',
		day: 'numeric',
		hour: '2-digit',
		minute: '2-digit'
	});
}

/**
 * Truncate a string to the given length with ellipsis.
 */
export function truncate(str: string, maxLen: number): string {
	if (str.length <= maxLen) return str;
	return str.slice(0, maxLen - 1) + '\u2026';
}
