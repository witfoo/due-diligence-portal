<script lang="ts">
	import { page } from '$app/state';
	import { authStore } from '$stores/authStore.svelte';

	let status = $derived(page.status);
	let message = $derived(page.error?.message ?? 'Something went wrong');
</script>

<main>
	<div class="box">
		<p class="status">{status}</p>
		<h1>{status === 404 ? 'Page not found' : 'Something went wrong'}</h1>
		<p class="detail">{message}</p>
		<a href={authStore.isAuthenticated ? '/documents' : '/login'} class="btn-primary">
			{authStore.isAuthenticated ? 'Back to Documents' : 'Sign In'}
		</a>
	</div>
</main>

<style>
	main { display: flex; justify-content: center; align-items: center; min-height: 100vh; }
	.box {
		text-align: center;
		max-width: 460px;
		padding: 2.5rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
	}
	.status { font-size: 3rem; font-weight: 200; color: var(--dd-text-secondary); margin: 0; }
	h1 { font-size: 1.5rem; font-weight: 400; margin: 0.5rem 0 1rem; }
	.detail { color: var(--dd-text-secondary); margin-bottom: 2rem; font-size: 0.875rem; }
	.btn-primary {
		display: inline-block;
		padding: 0.75rem 1.5rem;
		background: var(--dd-primary);
		color: #fff;
		text-decoration: none;
		font-size: 0.875rem;
	}
</style>
