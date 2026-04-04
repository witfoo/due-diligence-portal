<script lang="ts">
	import type { User } from '$types/api';
	import { api } from '$api/client';

	let users = $state<User[]>([]);
	let loading = $state(true);
	let showInvite = $state(false);
	let inviteEmail = $state('');
	let inviteRole = $state('investor');
	let inviteMessage = $state('');

	async function loadUsers() {
		loading = true;
		try {
			const res = await api.get<User[]>('/users');
			users = res.data ?? [];
		} catch {
			users = [];
		}
		loading = false;
	}

	async function sendInvite() {
		if (!inviteEmail.trim()) return;
		inviteMessage = '';
		try {
			const res = await api.post<{ token: string }>('/users/invite', {
				email: inviteEmail,
				role: inviteRole
			});
			if (res.data) {
				inviteMessage = `Invite sent. Token: ${res.data.token}`;
				inviteEmail = '';
				showInvite = false;
				await loadUsers();
			}
		} catch {
			inviteMessage = 'Failed to create invite.';
		}
	}

	function formatDate(iso: string | undefined): string {
		if (!iso) return 'Never';
		return new Date(iso).toLocaleDateString('en-US', {
			month: 'short', day: 'numeric', year: 'numeric'
		});
	}

	$effect(() => { loadUsers(); });
</script>

<h1>Users</h1>

<div class="actions">
	<button class="btn-primary" onclick={() => (showInvite = !showInvite)}>
		{showInvite ? 'Cancel' : 'Invite User'}
	</button>
</div>

{#if inviteMessage}
	<div class="message">{inviteMessage}</div>
{/if}

{#if showInvite}
	<div class="invite-form">
		<input type="email" placeholder="Email address" bind:value={inviteEmail} />
		<select bind:value={inviteRole}>
			<option value="investor">Investor</option>
			<option value="company_member">Company Member</option>
		</select>
		<button class="btn-primary" onclick={sendInvite}>Send Invite</button>
	</div>
{/if}

{#if loading}
	<p class="loading">Loading users...</p>
{:else if users.length === 0}
	<p class="empty">No users found.</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>Name</th>
				<th>Email</th>
				<th>Role</th>
				<th>Status</th>
				<th>Last Login</th>
			</tr>
		</thead>
		<tbody>
			{#each users as user}
				<tr>
					<td>{user.name}</td>
					<td>{user.email}</td>
					<td><span class="role-badge">{user.role.replace('_', ' ')}</span></td>
					<td>
						<span class="status" class:active={user.is_active} class:disabled={!user.is_active}>
							{user.is_active ? 'Active' : 'Disabled'}
						</span>
					</td>
					<td>{formatDate(user.last_login_at)}</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}

<style>
	h1 { font-weight: 400; font-size: 1.5rem; margin-bottom: 1.5rem; }

	.actions { margin-bottom: 1rem; }

	.btn-primary {
		padding: 0.5rem 1rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		cursor: pointer;
		font-size: 0.8125rem;
	}

	.message {
		padding: 0.75rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-primary);
		margin-bottom: 1rem;
		font-size: 0.8125rem;
		word-break: break-all;
	}

	.invite-form {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
		padding: 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
	}

	.invite-form input, .invite-form select {
		padding: 0.5rem 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.8125rem;
	}

	.invite-form input { flex: 1; }

	table { width: 100%; border-collapse: collapse; }

	th {
		text-align: left;
		padding: 0.5rem 0.75rem;
		background: var(--dd-surface);
		border-bottom: 1px solid var(--dd-border);
		font-size: 0.75rem;
		text-transform: uppercase;
		color: var(--dd-text-secondary);
	}

	td {
		padding: 0.5rem 0.75rem;
		border-bottom: 1px solid var(--dd-border);
		font-size: 0.8125rem;
	}

	.role-badge {
		padding: 0.125rem 0.5rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		border-radius: 2px;
		font-size: 0.75rem;
		text-transform: capitalize;
	}

	.status.active { color: var(--dd-success, #24a148); }
	.status.disabled { color: var(--dd-error, #da1e28); }

	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }
</style>
