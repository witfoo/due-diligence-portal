/**
 * API Client for the Due Diligence Portal.
 * Follows the WitFoo Analytics pattern: JWT auth, typed responses, error handling.
 */

const API_BASE = '/api/v1';
const ACCESS_KEY = 'dd_auth_token';
const REFRESH_KEY = 'dd_refresh_token';

export class ApiError extends Error {
	constructor(
		public status: number,
		public statusText: string,
		public body: unknown
	) {
		super(`API Error: ${status} ${statusText}`);
		this.name = 'ApiError';
	}
}

export interface ApiResponse<T> {
	success: boolean;
	message?: string;
	data?: T;
	meta?: {
		count: number;
		total?: number;
		page?: number;
		page_size?: number;
		has_more?: boolean;
	};
	timestamp: string;
}

export interface ApiErrorResponse {
	success: false;
	error: string;
	details?: Array<{
		code?: string;
		field?: string;
		message: string;
	}>;
	timestamp: string;
}

function getAuthToken(): string | null {
	if (typeof window === 'undefined') return null;
	return localStorage.getItem(ACCESS_KEY);
}

function getRefreshToken(): string | null {
	if (typeof window === 'undefined') return null;
	return localStorage.getItem(REFRESH_KEY);
}

function setAuthToken(token: string): void {
	if (typeof window === 'undefined') return;
	localStorage.setItem(ACCESS_KEY, token);
}

function setTokens(access: string, refresh?: string): void {
	if (typeof window === 'undefined') return;
	localStorage.setItem(ACCESS_KEY, access);
	if (refresh) localStorage.setItem(REFRESH_KEY, refresh);
}

function clearTokens(): void {
	if (typeof window === 'undefined') return;
	localStorage.removeItem(ACCESS_KEY);
	localStorage.removeItem(REFRESH_KEY);
}

function redirectToLogin(): void {
	if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
		window.location.href = '/login';
	}
}

/**
 * Attempt to obtain a fresh access token using the stored refresh token.
 * Returns true on success (the new access token is persisted).
 */
async function tryRefresh(): Promise<boolean> {
	const refresh = getRefreshToken();
	if (!refresh) return false;
	try {
		const resp = await fetch(`${API_BASE}/auth/refresh`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ refresh_token: refresh })
		});
		if (!resp.ok) return false;
		const data = await resp.json();
		const newToken = data?.data?.access_token;
		if (typeof newToken === 'string' && newToken) {
			setAuthToken(newToken);
			return true;
		}
	} catch {
		// fall through
	}
	return false;
}

async function request<T>(
	method: string,
	path: string,
	body?: unknown,
	options: RequestInit = {},
	isRetry = false
): Promise<ApiResponse<T>> {
	const url = `${API_BASE}${path}`;
	const headers: Record<string, string> = {
		...(options.headers as Record<string, string>)
	};

	const token = getAuthToken();
	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	if (body && !(body instanceof FormData)) {
		headers['Content-Type'] = 'application/json';
	}

	const response = await fetch(url, {
		method,
		headers,
		body: body instanceof FormData ? body : body ? JSON.stringify(body) : undefined,
		...options
	});

	if (response.status === 401) {
		// Try a one-shot token refresh before giving up (skip for the auth endpoints
		// themselves to avoid loops).
		const isAuthPath = path.startsWith('/auth/');
		if (!isRetry && !isAuthPath && (await tryRefresh())) {
			return request<T>(method, path, body, options, true);
		}
		clearTokens();
		redirectToLogin();
		throw new ApiError(401, 'Unauthorized', null);
	}

	if (!response.ok) {
		const errorBody = await response.json().catch(() => null);
		throw new ApiError(response.status, response.statusText, errorBody);
	}

	if (response.status === 204) {
		return { success: true, timestamp: new Date().toISOString() } as ApiResponse<T>;
	}

	return response.json();
}

/**
 * Download a file from an authenticated endpoint. Uses fetch (so the JWT is sent in
 * the Authorization header), reads the response as a Blob, and triggers a browser
 * download — a plain <a href> cannot attach the bearer token and would 401.
 */
async function download(path: string, fallbackName = 'download', isRetry = false): Promise<void> {
	if (typeof window === 'undefined') return;
	const token = getAuthToken();
	const resp = await fetch(`${API_BASE}${path}`, {
		headers: token ? { Authorization: `Bearer ${token}` } : {}
	});

	if (resp.status === 401) {
		if (!isRetry && (await tryRefresh())) {
			return download(path, fallbackName, true);
		}
		clearTokens();
		redirectToLogin();
		throw new ApiError(401, 'Unauthorized', null);
	}
	if (!resp.ok) {
		throw new ApiError(resp.status, resp.statusText, null);
	}

	// Prefer the server-provided filename from Content-Disposition.
	let name = fallbackName;
	const cd = resp.headers.get('Content-Disposition');
	const match = cd?.match(/filename="?([^"]+)"?/);
	if (match) name = match[1];

	const blob = await resp.blob();
	const objectUrl = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = objectUrl;
	a.download = name;
	document.body.appendChild(a);
	a.click();
	a.remove();
	URL.revokeObjectURL(objectUrl);
}

export const api = {
	get: <T>(path: string) => request<T>('GET', path),
	post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
	put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
	patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
	delete: <T>(path: string) => request<T>('DELETE', path),
	upload: <T>(path: string, formData: FormData) => request<T>('POST', path, formData),
	download,
	setToken: setAuthToken,
	setTokens,
	clearToken: clearTokens,
	clearTokens,
	getToken: getAuthToken
};
