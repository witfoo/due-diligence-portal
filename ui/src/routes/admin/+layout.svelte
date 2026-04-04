<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';

	let { children }: { children: Snippet } = $props();
	let currentPath = $derived(page.url.pathname);
</script>

<div class="admin-layout">
	<nav class="admin-nav">
		<h2>Admin</h2>
		<ul>
			<li><a href="/admin/users" class:active={currentPath.startsWith('/admin/users')}>Users</a></li>
			<li><a href="/admin/categories" class:active={currentPath.startsWith('/admin/categories')}>Categories</a></li>
			<li><a href="/admin/branding" class:active={currentPath.startsWith('/admin/branding')}>Branding</a></li>
			<li><a href="/admin/watermark" class:active={currentPath.startsWith('/admin/watermark')}>Watermark</a></li>
			<li><a href="/admin/nda" class:active={currentPath.startsWith('/admin/nda')}>NDA Templates</a></li>
			<li><a href="/admin/audit" class:active={currentPath.startsWith('/admin/audit')}>Audit Log</a></li>
		</ul>
	</nav>

	<main class="admin-content">
		{@render children()}
	</main>
</div>

<style>
	.admin-layout {
		display: flex;
		min-height: calc(100vh - 48px);
	}

	.admin-nav {
		width: 240px;
		background: var(--dd-sidebar);
		border-right: 1px solid var(--dd-border);
		padding: 1.5rem 0;
		flex-shrink: 0;
	}

	.admin-nav h2 {
		font-size: 0.875rem;
		font-weight: 600;
		text-transform: uppercase;
		color: var(--dd-text-secondary);
		padding: 0 1.5rem;
		margin-bottom: 1rem;
	}

	.admin-nav ul {
		list-style: none;
		padding: 0;
		margin: 0;
	}

	.admin-nav li a {
		display: block;
		padding: 0.625rem 1.5rem;
		color: var(--dd-text);
		text-decoration: none;
		font-size: 0.875rem;
		border-left: 3px solid transparent;
	}

	.admin-nav li a:hover {
		background: var(--dd-hover);
		border-left-color: var(--dd-primary);
	}

	.admin-nav li a.active {
		background: var(--dd-hover);
		border-left-color: var(--dd-primary);
		color: var(--dd-text);
	}

	.admin-content {
		flex: 1;
		padding: 2rem;
		max-width: 1000px;
	}
</style>
