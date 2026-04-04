<script lang="ts">
	import '../app.css';
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { authStore } from '$stores/authStore.svelte';
	import NavHeader from '$components/NavHeader.svelte';

	let { children }: { children: Snippet } = $props();

	// Public routes that don't show navigation.
	const publicRoutes = ['/', '/login'];
	let isPublicRoute = $derived(publicRoutes.includes(page.url.pathname));
	let hasToken = $derived(authStore.token !== null);
	let showNav = $derived(hasToken && !isPublicRoute);

	// Bootstrap auth on mount: restore user from stored token.
	$effect(() => {
		authStore.loadFromStorage();
		if (authStore.token && !authStore.user) {
			authStore.restore();
		}
	});
</script>

<svelte:head>
	<title>Due Diligence Portal</title>
</svelte:head>

{#if showNav}
	<NavHeader />
{/if}

<div class="app-content" class:has-nav={showNav}>
	{@render children()}
</div>

<style>
	.app-content {
		min-height: 100vh;
	}

	.app-content.has-nav {
		min-height: calc(100vh - 48px);
	}
</style>
