/**
 * Authentication store using Svelte 5 runes.
 * Follows the WitFoo Analytics pattern for reactive state management.
 */
import type { User } from '$types/api';

class AuthStore {
	user = $state<User | null>(null);
	token = $state<string | null>(null);
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
}

export const authStore = new AuthStore();
