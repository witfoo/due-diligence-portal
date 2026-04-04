/**
 * Authentication store using Svelte 5 runes.
 * Follows the WitFoo Analytics pattern for reactive state management.
 */
import type { User } from '$types/api';
import { api } from '$api/client';

class AuthStore {
	user = $state<User | null>(null);
	token = $state<string | null>(null);
	loading = $state(false);
	isAuthenticated = $derived(this.user !== null && this.token !== null);
	isAdmin = $derived(this.user?.role === 'admin');
	isCompanyMember = $derived(this.user?.role === 'company_member');
	isInvestor = $derived(this.user?.role === 'investor');

	setAuth(user: User, token: string) {
		this.user = user;
		this.token = token;
		if (typeof window !== 'undefined') {
			localStorage.setItem('dd_auth_token', token);
		}
	}

	clearAuth() {
		this.user = null;
		this.token = null;
		if (typeof window !== 'undefined') {
			localStorage.removeItem('dd_auth_token');
		}
	}

	loadFromStorage() {
		if (typeof window === 'undefined') return;
		const token = localStorage.getItem('dd_auth_token');
		if (token) {
			this.token = token;
		}
	}

	/**
	 * Restore user session from stored token by calling GET /auth/me.
	 * Call after loadFromStorage() to hydrate the user object.
	 */
	async restore(): Promise<boolean> {
		if (typeof window === 'undefined') return false;
		if (this.user) return true; // Already restored.

		const token = this.token ?? localStorage.getItem('dd_auth_token');
		if (!token) return false;

		this.token = token;
		this.loading = true;

		try {
			const resp = await api.get<{
				user_id: string;
				email: string;
				name: string;
				role: 'admin' | 'company_member' | 'investor';
			}>('/auth/me');

			if (resp.data) {
				this.user = {
					id: resp.data.user_id,
					email: resp.data.email,
					name: resp.data.name,
					role: resp.data.role,
					is_active: true,
					created_at: '',
					updated_at: ''
				};
				return true;
			}
		} catch {
			// Token expired or invalid — clear everything.
			this.clearAuth();
		} finally {
			this.loading = false;
		}
		return false;
	}
}

export const authStore = new AuthStore();
