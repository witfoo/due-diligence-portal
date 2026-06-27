<script lang="ts">
	import { api, ApiError } from '$api/client';
	import { authStore } from '$stores/authStore.svelte';
	import { page } from '$app/state';

	let token = $derived(page.url.searchParams.get('token') ?? '');
	let name = $state('');
	let password = $state('');
	let confirm = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		if (!token) {
			error = 'Missing or invalid invite token.';
			return;
		}
		if (password.length < 8) {
			error = 'Password must be at least 8 characters.';
			return;
		}
		if (password !== confirm) {
			error = 'Passwords do not match.';
			return;
		}

		loading = true;
		try {
			const resp = await api.post<{
				user: { id: string; email: string; name: string; role: string };
				access_token: string;
				refresh_token: string;
			}>('/auth/register', { token, name: name.trim(), password });

			if (resp.data) {
				const u = resp.data.user;
				authStore.setAuth(
					{
						id: u.id, email: u.email, name: u.name,
						role: u.role as 'admin' | 'company_member' | 'investor',
						is_active: true, created_at: '', updated_at: ''
					},
					resp.data.access_token,
					resp.data.refresh_token
				);
				window.location.href = '/documents';
			}
		} catch (err) {
			if (err instanceof ApiError) {
				if (err.status === 404) error = 'This invite link is invalid.';
				else if (err.status === 409) error = 'This invite has already been used.';
				else if (err.status === 400) error = 'This invite link has expired or is invalid.';
				else error = 'Registration failed. Please try again.';
			} else {
				error = 'Unable to connect to the server.';
			}
		} finally {
			loading = false;
		}
	}
</script>

<main>
	<div class="register-container">
		<h1>Create Your Account</h1>

		{#if !token}
			<div class="error-msg">No invite token found. Please use the link from your invitation email.</div>
		{:else}
			{#if error}<div class="error-msg">{error}</div>{/if}
			<form onsubmit={handleSubmit}>
				<div class="field">
					<label for="name">Full Name</label>
					<input id="name" type="text" bind:value={name} required autocomplete="name" />
				</div>
				<div class="field">
					<label for="password">Password</label>
					<input id="password" type="password" bind:value={password} required
						autocomplete="new-password" placeholder="At least 8 characters" />
				</div>
				<div class="field">
					<label for="confirm">Confirm Password</label>
					<input id="confirm" type="password" bind:value={confirm} required autocomplete="new-password" />
				</div>
				<button type="submit" class="btn-primary" disabled={loading}>
					{loading ? 'Creating account…' : 'Create Account'}
				</button>
			</form>
		{/if}
	</div>
</main>

<style>
	main { display: flex; justify-content: center; align-items: center; min-height: 100vh; }
	.register-container {
		width: 100%;
		max-width: 400px;
		padding: 2rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
	}
	h1 { font-size: 1.5rem; font-weight: 400; margin-bottom: 2rem; }
	.error-msg {
		padding: 0.75rem 1rem;
		background: #da1e2815;
		border: 1px solid var(--dd-error, #da1e28);
		color: var(--dd-error, #da1e28);
		font-size: 0.875rem;
		margin-bottom: 1.5rem;
	}
	.field { margin-bottom: 1.5rem; }
	label { display: block; font-size: 0.875rem; color: var(--dd-text-secondary); margin-bottom: 0.5rem; }
	input {
		width: 100%;
		padding: 0.75rem 1rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.875rem;
	}
	.btn-primary {
		width: 100%;
		padding: 0.875rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		font-size: 1rem;
		cursor: pointer;
	}
	.btn-primary:disabled { opacity: 0.5; cursor: wait; }
</style>
