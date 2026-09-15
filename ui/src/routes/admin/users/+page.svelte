<script lang="ts">
	import { tick } from 'svelte';
	import type { User, InviteToken, SetPasswordResponse } from '$types/api';
	import { api, apiErrorMessage } from '$api/client';
	import { authStore } from '$stores/authStore.svelte';
	import { PASSWORD_MAX_BYTES, PASSWORD_MIN_LENGTH, validateNewPassword } from '$lib/utils/password';

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

	// Set-password form state. Only one inline form is open at a time (keyed by user id).
	let passwordUserId = $state<string | null>(null);
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let passwordError = $state('');
	let passwordSaving = $state(false);
	let passwordMessage = $state('');
	// The user row the success message belongs to; the confirmation is shown beside it.
	let passwordMessageUserId = $state<string | null>(null);

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

	/** Close the set-password form and drop every password value from state. */
	function resetPasswordForm() {
		passwordUserId = null;
		currentPassword = '';
		newPassword = '';
		confirmPassword = '';
		passwordError = '';
	}

	/**
	 * Move keyboard focus back to a row's Set Password button. Closing the form
	 * removes the button that had focus, which would otherwise leave focus on the
	 * page body.
	 */
	async function focusSetPasswordButton(userId: string) {
		await tick();
		document.getElementById(`set-password-${userId}`)?.focus();
	}

	async function cancelPasswordForm(userId: string) {
		resetPasswordForm();
		await focusSetPasswordButton(userId);
	}

	function togglePasswordForm(user: User) {
		if (passwordSaving) return;
		const opening = passwordUserId !== user.id;
		resetPasswordForm();
		if (opening) {
			passwordMessage = '';
			passwordMessageUserId = null;
			passwordUserId = user.id;
		}
	}

	async function submitPassword(e: Event, user: User, isSelf: boolean) {
		e.preventDefault();
		if (passwordSaving) return;
		passwordError = '';
		passwordMessage = '';
		passwordMessageUserId = null;

		// Same order as the server: validate the new password, then require re-authentication.
		const invalid = validateNewPassword(newPassword, confirmPassword);
		if (invalid) {
			passwordError = invalid;
			return;
		}
		if (isSelf && !currentPassword) {
			passwordError = 'Current password is required.';
			return;
		}

		const body: { password: string; current_password?: string } = { password: newPassword };
		if (isSelf) body.current_password = currentPassword;

		passwordSaving = true;
		let updated = false;
		try {
			const res = await api.put<SetPasswordResponse>(`/users/${user.id}/password`, body);
			// Changing your own password revokes your previous refresh token; keep the
			// session alive with the freshly minted pair.
			if (isSelf && res.data?.access_token) {
				api.setTokens(res.data.access_token, res.data.refresh_token);
			}
			resetPasswordForm();
			passwordMessage = isSelf
				? 'Your password has been updated.'
				: `Password updated for ${user.email}. Share it with them through a secure channel.`;
			passwordMessageUserId = user.id;
			updated = true;
		} catch (err) {
			passwordError = apiErrorMessage(err, `Failed to set password for ${user.email}.`);
		} finally {
			passwordSaving = false;
		}
		// Only after passwordSaving clears: a disabled button cannot take focus.
		if (updated) await focusSetPasswordButton(user.id);
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
<!-- Always in the DOM so screen readers announce password updates when the text
     changes. Sighted users see the confirmation beside the user's row. -->
<div class="visually-hidden" role="status">{passwordMessage}</div>
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
						<div class="row-actions">
							<button
								id="set-password-{user.id}"
								class="btn-small"
								aria-expanded={passwordUserId === user.id}
								aria-controls="password-form-{user.id}"
								disabled={passwordSaving}
								onclick={() => togglePasswordForm(user)}
							>Set Password</button>
							<!-- Admins cannot disable their own account. -->
							{#if !isSelf}
								{#if user.is_active}
									<button class="btn-small danger" onclick={() => setActive(user, false)}>Disable</button>
								{:else}
									<button class="btn-small" onclick={() => setActive(user, true)}>Enable</button>
								{/if}
							{/if}
						</div>
					</td>
				</tr>
				{#if passwordUserId === user.id}
					<tr class="password-row">
						<td colspan="6">
							<form
								id="password-form-{user.id}"
								class="password-form"
								onsubmit={(e) => submitPassword(e, user, isSelf)}
							>
								<p class="password-title">
									{isSelf ? 'Change your password' : `Set a new password for ${user.email}`}
								</p>
								{#if isSelf}
									<div class="field">
										<label for="pw-current-{user.id}">Current password</label>
										<input
											id="pw-current-{user.id}"
											name="current_password"
											type="password"
											autocomplete="current-password"
											aria-label="Current password for {user.email}"
											bind:value={currentPassword}
										/>
									</div>
								{/if}
								<div class="field">
									<label for="pw-new-{user.id}">New password</label>
									<input
										id="pw-new-{user.id}"
										name="new_password"
										type="password"
										autocomplete="new-password"
										aria-label="New password for {user.email}"
										aria-describedby="pw-hint-{user.id}"
										bind:value={newPassword}
									/>
								</div>
								<div class="field">
									<label for="pw-confirm-{user.id}">Confirm new password</label>
									<input
										id="pw-confirm-{user.id}"
										name="confirm_password"
										type="password"
										autocomplete="new-password"
										aria-label="Confirm new password for {user.email}"
										bind:value={confirmPassword}
									/>
								</div>
								<div class="form-actions">
									<button class="btn-primary" type="submit" disabled={passwordSaving}>
										{passwordSaving ? 'Saving…' : 'Save Password'}
									</button>
									<button
										class="btn-small"
										type="button"
										disabled={passwordSaving}
										onclick={() => cancelPasswordForm(user.id)}
									>Cancel</button>
								</div>
								<p class="hint form-hint" id="pw-hint-{user.id}">
									At least {PASSWORD_MIN_LENGTH} characters and no more than {PASSWORD_MAX_BYTES} bytes.
								</p>
								{#if passwordError}<div class="error-msg form-error" role="alert">{passwordError}</div>{/if}
							</form>
						</td>
					</tr>
				{:else if passwordMessage && passwordMessageUserId === user.id}
					<tr class="password-row">
						<td colspan="6"><div class="message password-message">{passwordMessage}</div></td>
					</tr>
				{/if}
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
	.btn-small:disabled, .btn-primary:disabled { opacity: 0.5; cursor: wait; }
	.row-actions { display: flex; flex-wrap: wrap; gap: 0.5rem; }
	.password-row td { padding: 0.75rem; background: var(--dd-background); }
	.password-form {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-end;
		gap: 0.5rem 0.75rem;
		padding: 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
	}
	.password-title { flex-basis: 100%; margin: 0; font-size: 0.8125rem; color: var(--dd-text); }
	.password-form .field { display: flex; flex-direction: column; gap: 0.25rem; flex: 1 1 12rem; min-width: 0; }
	.password-form label { font-size: 0.75rem; color: var(--dd-text-secondary); }
	.password-form input {
		padding: 0.5rem 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.8125rem;
	}
	.password-form .form-actions { display: flex; align-items: center; gap: 0.5rem; }
	.password-form .form-hint, .password-form .form-error { flex-basis: 100%; margin: 0; }
	.password-message { margin: 0; }
	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}
	.link {
		background: none;
		border: none;
		color: var(--dd-primary);
		cursor: pointer;
		text-decoration: underline;
		font-size: inherit;
	}
	.invites { margin-bottom: 2rem; }
	.invites h2 { font-weight: 400; font-size: 1.125rem; margin-bottom: 0.25rem; }
	.hint { font-size: 0.8125rem; color: var(--dd-text-secondary); margin-bottom: 0.75rem; }
	.link-cell { font-family: monospace; font-size: 0.75rem; word-break: break-all; color: var(--dd-primary); }
	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }
</style>
