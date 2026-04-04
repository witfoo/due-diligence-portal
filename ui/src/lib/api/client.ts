/**
 * API Client for the Due Diligence Portal.
 * Follows the WitFoo Analytics pattern: JWT auth, typed responses, error handling.
 */

const API_BASE = '/api/v1';

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
	return localStorage.getItem('dd_auth_token');
}

function setAuthToken(token: string): void {
	if (typeof window === 'undefined') return;
	localStorage.setItem('dd_auth_token', token);
}

function clearAuthToken(): void {
	if (typeof window === 'undefined') return;
	localStorage.removeItem('dd_auth_token');
}

async function request<T>(
	method: string,
	path: string,
	body?: unknown,
	options: RequestInit = {}
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
		clearAuthToken();
		if (typeof window !== 'undefined') {
			window.location.href = '/login';
		}
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

export const api = {
	get: <T>(path: string) => request<T>('GET', path),
	post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
	put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
	patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
	delete: <T>(path: string) => request<T>('DELETE', path),
	upload: <T>(path: string, formData: FormData) => request<T>('POST', path, formData),
	setToken: setAuthToken,
	clearToken: clearAuthToken,
	getToken: getAuthToken
};
