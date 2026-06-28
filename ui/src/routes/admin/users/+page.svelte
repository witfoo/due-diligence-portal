<script lang="ts">
	import type { User, InviteToken } from '$types/api';
	import { api } from '$api/client';
	import { authStore } from '$stores/authStore.svelte';

	let users = $state<User[]>([]);
	let invites = $state<InviteToken[]>([]);
	let loading = $state(true);
	let loadError = $state('');
	let showInvite = $state(false);
	let inviteEmail = $state('');
	let inviteRole = $state('investor');
	let inviteMessage = $state('');
	let inviteLink = $state('');
	let actionError = $state('');

	async function loadUsers() {
		loading = true;
		loadError = '';
		try {
			const res = await api.get<User[]>('/users');
			users = res.data ?? [];
		} catch {
			loadError = 'Failed to load users.';
			users = [];
		}
		loading = false;
	}

	async function loadInvites() {
		try {
			const res = await api.get<InviteToken[]>('/users/invites');
			invites = res.data ?? [];
		} catch {
			invites = [];
		}
	}

	function registerLink(token: string): string {
		return `${window.location.origin}/register?token=${token}`;
	}

	async function revokeInvite(id: string) {
		actionError = '';
		try {
			await api.delete(`/users/invites/${id}`);
			await loadInvites();
		} catch {
			actionError = 'Failed to revoke invitation.';
		}
	}

	async function sendInvite(e: Event) {
		e.preventDefault();
		if (!inviteEmail.trim()) return;
		inviteMessage = '';
		inviteLink = '';
		try {
			const res = await api.post<{ token: string }>('/users/invite', {
				email: inviteEmail,
				role: inviteRole
			});
			if (res.data) {
				inviteMessage = 'Invitation created. Share this registration link securely:';
				inviteLink = `${window.location.origin}/register?token=${res.data.token}`;
				inviteEmail = '';
				showInvite = false;
				await loadInvites();
				await loadUsers();
			}
		} catch {
			inviteMessage = 'Failed to create invite.';
		}
	}

	async function setActive(user: User, active: boolean) {
		actionError = '';
		try {
			await api.put(`/users/${user.id}`, { is_active: active });
			await loadUsers();
		} catch {
			actionError = `Failed to ${active ? 'enable' : 'disable'} ${user.email}.`;
		}
	}

	async function changeRole(user: User, role: string) {
		if (role === user.role) return;
		actionError = '';
		try {
			await api.put(`/users/${user.id}`, { role });
			await loadUsers();
		} catch {
			actionError = `Failed to change role for ${user.email}.`;
		}
	}

	function formatDate(iso: string | undefined): string {
		if (!iso) return 'Never';
		return new Date(iso).toLocaleDateString('en-US', {
			month: 'short', day: 'numeric', year: 'numeric'
		});
	}

	$effect(() => { loadUsers(); loadInvites(); });
</script>

<h1>Users</h1>

<div class="actions">
	<button class="btn-primary" onclick={() => (showInvite = !showInvite)}>
		{showInvite ? 'Cancel' : 'Invite User'}
	</button>
</div>

{#if inviteMessage}
	<div class="message">
		{inviteMessage}
		{#if inviteLink}<div class="invite-link">{inviteLink}</div>{/if}
	</div>
{/if}
{#if actionError}<div class="error-msg">{actionError}</div>{/if}

{#if showInvite}
	<form class="invite-form" onsubmit={sendInvite}>
		<input type="email" placeholder="Email address" bind:value={inviteEmail} required aria-label="Invite email" />
		<select bind:value={inviteRole} aria-label="Invite role">
			<option value="investor">Investor</option>
			<option value="company_member">Company Member</option>
		</select>
		<button class="btn-primary" type="submit">Send Invite</button>
	</form>
{/if}

{#if invites.length > 0}
	<section class="invites">
		<h2>Pending Invitations ({invites.length})</h2>
		<p class="hint">These people have been invited but haven't registered yet. Share the registration link with them.</p>
		<table>
			<thead>
				<tr><th>Email</th><th>Role</th><th>Expires</th><th>Registration link</th><th>Actions</th></tr>
			</thead>
			<tbody>
				{#each invites as inv}
					<tr>
						<td>{inv.email}</td>
						<td><span class="role-badge">{inv.role.replace('_', ' ')}</span></td>
						<td>{formatDate(inv.expires_at)}</td>
						<td class="link-cell">{registerLink(inv.token)}</td>
						<td><button class="btn-small danger" onclick={() => revokeInvite(inv.id)}>Revoke</button></td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>
{/if}

{#if loading}
	<p class="loading">Loading users...</p>
{:else if loadError}
	<p class="error-msg">{loadError} <button class="link" onclick={loadUsers}>Retry</button></p>
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
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#each users as user}
				{@const isSelf = user.id === authStore.user?.id}
				<tr>
					<td>{user.name}</td>
					<td>{user.email}</td>
					<td>
						{#if isSelf}
							<span class="role-badge">{user.role.replace('_', ' ')}</span>
						{:else}
							<select
								class="role-select"
								value={user.role}
								aria-label="Role for {user.email}"
								onchange={(e) => changeRole(user, (e.currentTarget as HTMLSelectElement).value)}
							>
								<option value="investor">investor</option>
								<option value="company_member">company member</option>
								<option value="admin">admin</option>
							</select>
						{/if}
					</td>
					<td>
						<span class="status" class:active={user.is_active} class:disabled={!user.is_active}>
							{user.is_active ? 'Active' : 'Disabled'}
						</span>
					</td>
					<td>{formatDate(user.last_login_at)}</td>
					<td>
						{#if isSelf}
							<span class="muted">—</span>
						{:else if user.is_active}
							<button class="btn-small danger" onclick={() => setActive(user, false)}>Disable</button>
						{:else}
							<button class="btn-small" onclick={() => setActive(user, true)}>Enable</button>
						{/if}
					</td>
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
	}
	.invite-link {
		margin-top: 0.5rem;
		font-family: monospace;
		word-break: break-all;
		color: var(--dd-primary);
	}
	.error-msg {
		padding: 0.5rem 0.75rem;
		background: #da1e2815;
		border: 1px solid var(--dd-error, #da1e28);
		color: var(--dd-error, #da1e28);
		font-size: 0.8125rem;
		margin-bottom: 1rem;
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
	.role-select {
		padding: 0.25rem 0.5rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.75rem;
	}
	.status.active { color: var(--dd-success, #24a148); }
	.status.disabled { color: var(--dd-error, #da1e28); }
	.btn-small {
		padding: 0.25rem 0.625rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		cursor: pointer;
		font-size: 0.75rem;
	}
	.btn-small.danger:hover { color: var(--dd-error, #da1e28); border-color: var(--dd-error, #da1e28); }
	.link {
		background: none;
		border: none;
		color: var(--dd-primary);
		cursor: pointer;
		text-decoration: underline;
		font-size: inherit;
	}
	.muted { color: var(--dd-text-secondary); }
	.invites { margin-bottom: 2rem; }
	.invites h2 { font-weight: 400; font-size: 1.125rem; margin-bottom: 0.25rem; }
	.hint { font-size: 0.8125rem; color: var(--dd-text-secondary); margin-bottom: 0.75rem; }
	.link-cell { font-family: monospace; font-size: 0.75rem; word-break: break-all; color: var(--dd-primary); }
	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }
</style>
