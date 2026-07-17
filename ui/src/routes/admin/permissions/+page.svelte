<script lang="ts">
	import type { AccessGrant, Category, Document, User } from '$types/api';
	import { api } from '$api/client';

	let users = $state<User[]>([]);
	let categories = $state<Category[]>([]);
	let documents = $state<Document[]>([]);
	let grants = $state<AccessGrant[]>([]);
	let selectedUserId = $state('');
	let loadingGrants = $state(false);
	let message = $state('');

	let showGrantForm = $state(false);
	let resourceType = $state<'category' | 'document'>('category');
	let resourceId = $state('');
	let accessLevel = $state<'view' | 'download' | 'upload' | 'manage'>('download');
	let expiresOn = $state('');
	let granting = $state(false);

	let selectedUser = $derived(users.find((u) => u.id === selectedUserId));

	async function loadUsers() {
		try {
			const res = await api.get<User[]>('/users');
			// Investors first: they are the only role gated by grants.
			users = (res.data ?? []).sort((a, b) =>
				a.role === b.role ? a.name.localeCompare(b.name) : a.role === 'investor' ? -1 : 1
			);
		} catch {
			users = [];
		}
	}

	// The categories endpoint returns a nested tree; flatten for the picker.
	function flattenCategories(tree: Category[], depth = 0): Category[] {
		const out: Category[] = [];
		for (const c of tree) {
			out.push({ ...c, name: `${'— '.repeat(depth)}${c.name}` });
			if (c.children?.length) out.push(...flattenCategories(c.children, depth + 1));
		}
		return out;
	}

	async function loadResources() {
		try {
			const res = await api.get<Category[]>('/categories');
			categories = flattenCategories(res.data ?? []);
		} catch {
			categories = [];
		}
		try {
			const res = await api.get<Document[]>('/documents?limit=100');
			documents = res.data ?? [];
		} catch {
			documents = [];
		}
	}

	async function loadGrants() {
		if (!selectedUserId) {
			grants = [];
			return;
		}
		loadingGrants = true;
		try {
			const res = await api.get<AccessGrant[]>(`/permissions/user/${selectedUserId}`);
			grants = res.data ?? [];
		} catch {
			grants = [];
		}
		loadingGrants = false;
	}

	function onUserChange() {
		message = '';
		loadGrants();
	}

	async function grantAccess() {
		if (!selectedUserId || !resourceId) return;
		message = '';
		granting = true;
		try {
			const body: Record<string, unknown> = {
				user_id: selectedUserId,
				resource_type: resourceType,
				resource_id: resourceId,
				access_level: accessLevel
			};
			// The date input yields YYYY-MM-DD; grant until the end of that day (UTC).
			if (expiresOn) body.expires_at = `${expiresOn}T23:59:59Z`;
			await api.post('/permissions', body);
			resourceId = '';
			expiresOn = '';
			showGrantForm = false;
			message = 'Access granted.';
			await loadGrants();
		} catch {
			message = 'Failed to grant access.';
		}
		granting = false;
	}

	async function revokeGrant(grant: AccessGrant) {
		message = '';
		try {
			await api.delete(`/permissions/${grant.id}`);
			message = 'Grant revoked.';
			await loadGrants();
		} catch {
			message = 'Failed to revoke grant.';
		}
	}

	function isExpired(grant: AccessGrant): boolean {
		return !!grant.expires_at && new Date(grant.expires_at).getTime() < Date.now();
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleDateString('en-US', {
			month: 'short', day: 'numeric', year: 'numeric'
		});
	}

	$effect(() => {
		loadUsers();
		loadResources();
	});
</script>

<h1>Access Grants</h1>
<p class="hint">
	By default (<code>DD_INVESTOR_ACCESS=all</code>) every investor sees every document, and
	grants have no effect. Set <code>DD_INVESTOR_ACCESS=granted</code> on the server to
	restrict investors to documents they hold a grant for — either on the document itself
	or on its category. Admins and company members always see everything.
</p>

<div class="user-picker">
	<label for="grant-user">User</label>
	<select id="grant-user" bind:value={selectedUserId} onchange={onUserChange}>
		<option value="" disabled>Select a user…</option>
		{#each users as user}
			<option value={user.id}>{user.name} ({user.email}) — {user.role}</option>
		{/each}
	</select>
</div>

{#if message}
	<div class="message">{message}</div>
{/if}

{#if selectedUserId}
	{#if selectedUser && selectedUser.role !== 'investor'}
		<p class="staff-note">
			{selectedUser.name} is {selectedUser.role === 'admin' ? 'an admin' : 'a company member'}
			and can already access all documents; grants only affect investors.
		</p>
	{/if}

	<div class="actions">
		<button class="btn-primary" onclick={() => (showGrantForm = !showGrantForm)}>
			{showGrantForm ? 'Cancel' : 'Grant Access'}
		</button>
	</div>

	{#if showGrantForm}
		<div class="grant-form">
			<label>
				Resource type
				<select bind:value={resourceType} onchange={() => (resourceId = '')}>
					<option value="category">Category (all documents in it)</option>
					<option value="document">Single document</option>
				</select>
			</label>
			<label>
				{resourceType === 'category' ? 'Category' : 'Document'}
				<select bind:value={resourceId}>
					<option value="" disabled>Select…</option>
					{#if resourceType === 'category'}
						{#each categories as cat}
							<option value={cat.id}>{cat.name} ({cat.document_count ?? 0} documents)</option>
						{/each}
					{:else}
						{#each documents as doc}
							<option value={doc.id}>{doc.name}</option>
						{/each}
					{/if}
				</select>
			</label>
			<label>
				Access level
				<select bind:value={accessLevel}>
					<option value="view">View only</option>
					<option value="download">View + download</option>
					<option value="upload">View + download + upload</option>
					<option value="manage">Full manage</option>
				</select>
			</label>
			<label>
				Expires (optional)
				<input type="date" bind:value={expiresOn} />
			</label>
			<button class="btn-primary" onclick={grantAccess} disabled={!resourceId || granting}>
				{granting ? 'Granting…' : 'Grant Access'}
			</button>
		</div>
	{/if}

	{#if loadingGrants}
		<p class="loading">Loading grants...</p>
	{:else if grants.length === 0}
		<p class="empty">
			No grants for this user{selectedUser?.role === 'investor'
				? ' — they cannot see any documents yet.'
				: '.'}
		</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>Scope</th>
					<th>Resource</th>
					<th>Level</th>
					<th>Expires</th>
					<th>Granted</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				{#each grants as grant}
					<tr class:expired={isExpired(grant)}>
						<td class="cap">{grant.resource_type}</td>
						<td>{grant.resource_name || grant.resource_id}</td>
						<td class="cap">{grant.access_level}</td>
						<td>
							{#if grant.expires_at}
								{formatDate(grant.expires_at)}
								{#if isExpired(grant)}<span class="expired-tag">expired</span>{/if}
							{:else}
								Never
							{/if}
						</td>
						<td>{formatDate(grant.created_at)}</td>
						<td>
							<button class="btn-danger" onclick={() => revokeGrant(grant)}>Revoke</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
{/if}

<style>
	h1 { font-weight: 400; font-size: 1.5rem; margin-bottom: 0.75rem; }

	.hint { font-size: 0.8125rem; color: var(--dd-text-secondary); margin-bottom: 1.5rem; max-width: 48rem; }

	.user-picker {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		margin-bottom: 1rem;
		max-width: 32rem;
	}

	.user-picker label { font-size: 0.75rem; color: var(--dd-text-secondary); }

	select, input[type='date'] {
		padding: 0.5rem 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.8125rem;
	}

	.staff-note { font-size: 0.8125rem; color: var(--dd-text-secondary); margin-bottom: 1rem; }

	.actions { display: flex; gap: 0.5rem; margin-bottom: 1rem; }

	.btn-primary {
		padding: 0.5rem 1rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		cursor: pointer;
		font-size: 0.8125rem;
	}

	.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

	.btn-danger {
		padding: 0.25rem 0.75rem;
		background: transparent;
		color: var(--dd-danger, #da1e28);
		border: 1px solid var(--dd-danger, #da1e28);
		cursor: pointer;
		font-size: 0.75rem;
	}

	.message {
		padding: 0.75rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-primary);
		margin-bottom: 1rem;
		font-size: 0.8125rem;
	}

	.grant-form {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		margin-bottom: 1.5rem;
		padding: 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		max-width: 32rem;
	}

	.grant-form label {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		font-size: 0.75rem;
		color: var(--dd-text-secondary);
	}

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

	tr.expired td { color: var(--dd-text-secondary); }

	.expired-tag {
		margin-left: 0.5rem;
		color: var(--dd-danger, #da1e28);
		font-size: 0.6875rem;
		text-transform: uppercase;
	}

	.cap { text-transform: capitalize; }
	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }
</style>
