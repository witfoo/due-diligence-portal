<script lang="ts">
	import { page } from '$app/state';
	import { authStore } from '$stores/authStore.svelte';

	let currentPath = $derived(page.url.pathname);
	let showAnalytics = $derived(authStore.isAdmin || authStore.isCompanyMember);
	let showAdmin = $derived(authStore.isAdmin);
	let menuOpen = $state(false);

	interface NavItem {
		href: string;
		label: string;
	}

	let mainNav = $derived<NavItem[]>([
		{ href: '/documents', label: 'Documents' },
		{ href: '/qa', label: 'Q&A' },
		...(showAnalytics ? [{ href: '/analytics', label: 'Analytics' }] : []),
		...(showAdmin ? [{ href: '/admin/users', label: 'Admin' }] : [])
	]);

	function isActive(href: string): boolean {
		if (href.startsWith('/admin')) return currentPath.startsWith('/admin');
		return currentPath.startsWith(href);
	}

	function handleLogout() {
		authStore.clearAuth();
		window.location.href = '/login';
	}

	function handleClickOutside(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (menuOpen && !target.closest('.nav-user')) {
			menuOpen = false;
		}
	}
</script>

<svelte:window onclick={handleClickOutside} />

<header class="nav-header">
	<div class="nav-left">
		<a href="/documents" class="nav-brand">
			{authStore.user?.name ? 'Due Diligence Portal' : 'Portal'}
		</a>
		<nav class="nav-links">
			{#each mainNav as item}
				<a href={item.href} class="nav-link" class:active={isActive(item.href)}>
					{item.label}
				</a>
			{/each}
		</nav>
	</div>

	<div class="nav-user">
		<button class="user-trigger" onclick={() => (menuOpen = !menuOpen)}>
			<span class="user-name">{authStore.user?.name ?? authStore.user?.email ?? 'User'}</span>
			<span class="user-caret">{menuOpen ? '\u25B4' : '\u25BE'}</span>
		</button>

		{#if menuOpen}
			<div class="user-menu">
				<div class="menu-info">
					<span class="info-name">{authStore.user?.name}</span>
					<span class="info-email">{authStore.user?.email}</span>
					<span class="info-role">{authStore.user?.role?.replace('_', ' ')}</span>
				</div>
				<div class="menu-divider"></div>
				<button class="menu-item" onclick={handleLogout}>Sign Out</button>
			</div>
		{/if}
	</div>
</header>

<style>
	.nav-header {
		height: 48px;
		background: var(--dd-header);
		border-bottom: 1px solid var(--dd-border);
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 1rem;
		position: sticky;
		top: 0;
		z-index: 100;
	}

	.nav-left {
		display: flex;
		align-items: center;
		gap: 0;
	}

	.nav-brand {
		color: var(--dd-text);
		font-size: 0.875rem;
		font-weight: 600;
		text-decoration: none;
		padding-right: 1.5rem;
		margin-right: 0.5rem;
		border-right: 1px solid var(--dd-border);
		white-space: nowrap;
	}

	.nav-links {
		display: flex;
		align-items: center;
	}

	.nav-link {
		color: var(--dd-text-secondary);
		text-decoration: none;
		font-size: 0.875rem;
		padding: 0 0.875rem;
		height: 48px;
		display: inline-flex;
		align-items: center;
		border-bottom: 2px solid transparent;
		transition: color 0.15s, border-color 0.15s;
	}

	.nav-link:hover {
		color: var(--dd-text);
		background: var(--dd-hover);
		text-decoration: none;
	}

	.nav-link.active {
		color: var(--dd-text);
		border-bottom-color: var(--dd-primary);
	}

	.nav-user {
		position: relative;
	}

	.user-trigger {
		background: none;
		border: none;
		color: var(--dd-text-secondary);
		cursor: pointer;
		font-size: 0.8125rem;
		padding: 0.375rem 0.5rem;
		display: flex;
		align-items: center;
		gap: 0.375rem;
	}

	.user-trigger:hover {
		color: var(--dd-text);
	}

	.user-caret {
		font-size: 0.625rem;
	}

	.user-menu {
		position: absolute;
		top: 100%;
		right: 0;
		margin-top: 0.25rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		min-width: 220px;
		z-index: 200;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
	}

	.menu-info {
		padding: 0.75rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.info-name {
		font-size: 0.875rem;
		color: var(--dd-text);
		font-weight: 500;
	}

	.info-email {
		font-size: 0.75rem;
		color: var(--dd-text-secondary);
	}

	.info-role {
		font-size: 0.6875rem;
		color: var(--dd-text-secondary);
		text-transform: capitalize;
		margin-top: 0.25rem;
	}

	.menu-divider {
		height: 1px;
		background: var(--dd-border);
	}

	.menu-item {
		display: block;
		width: 100%;
		padding: 0.625rem 1rem;
		background: none;
		border: none;
		color: var(--dd-text);
		font-size: 0.8125rem;
		cursor: pointer;
		text-align: left;
	}

	.menu-item:hover {
		background: var(--dd-hover);
		color: var(--dd-error, #da1e28);
	}

	@media (max-width: 640px) {
		.nav-brand {
			display: none;
		}

		.nav-link {
			padding: 0 0.5rem;
			font-size: 0.8125rem;
		}
	}
</style>
