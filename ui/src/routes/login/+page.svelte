<script lang="ts">
	import { api } from '$api/client';
	import { authStore } from '$stores/authStore.svelte';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleLogin(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;

		try {
			const resp = await api.post<{
				user: { id: string; email: string; name: string; role: string };
				access_token: string;
				refresh_token: string;
			}>('/auth/login', { email, password });

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
		} catch (err: unknown) {
			if (err && typeof err === 'object' && 'status' in err) {
				const apiErr = err as { status: number };
				if (apiErr.status === 401) {
					error = 'Invalid email or password.';
				} else if (apiErr.status === 403) {
					error = 'Account is disabled. Contact your administrator.';
				} else {
					error = 'An error occurred. Please try again.';
				}
			} else {
				error = 'Unable to connect to the server.';
			}
		} finally {
			loading = false;
		}
	}
</script>

<main>
	<div class="login-container">
		<h1>Sign In</h1>

		{#if error}
			<div class="error-msg">{error}</div>
		{/if}

		<form onsubmit={handleLogin}>
			<div class="field">
				<label for="email">Email</label>
				<input id="email" type="email" placeholder="Enter your email"
					bind:value={email} required autocomplete="email" />
			</div>
			<div class="field">
				<label for="password">Password</label>
				<input id="password" type="password" placeholder="Enter your password"
					bind:value={password} required autocomplete="current-password" />
			</div>
			<button type="submit" class="btn-primary" disabled={loading}>
				{loading ? 'Signing in...' : 'Sign In'}
			</button>
		</form>
	</div>
</main>

<style>
	main {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 100vh;
	}

	.login-container {
		width: 100%;
		max-width: 400px;
		padding: 2rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
	}

	h1 {
		font-size: 1.75rem;
		font-weight: 400;
		margin-bottom: 2rem;
	}

	.error-msg {
		padding: 0.75rem 1rem;
		background: #da1e2815;
		border: 1px solid var(--dd-error, #da1e28);
		color: var(--dd-error, #da1e28);
		font-size: 0.875rem;
		margin-bottom: 1.5rem;
	}

	.field {
		margin-bottom: 1.5rem;
	}

	label {
		display: block;
		font-size: 0.875rem;
		color: var(--dd-text-secondary);
		margin-bottom: 0.5rem;
	}

	input {
		width: 100%;
		padding: 0.75rem 1rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.875rem;
	}

	input:focus {
		outline: 2px solid var(--dd-primary);
		outline-offset: -2px;
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

	.btn-primary:hover {
		background: #0353e9;
	}

	.btn-primary:disabled {
		opacity: 0.5;
		cursor: wait;
	}
</style>
