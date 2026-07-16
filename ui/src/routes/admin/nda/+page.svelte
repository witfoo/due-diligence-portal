<script lang="ts">
	import type { NDATemplate, NDASignature, NDAExemption, User } from '$types/api';
	import { api } from '$api/client';

	let templates = $state<NDATemplate[]>([]);
	let signatures = $state<NDASignature[]>([]);
	let exemptions = $state<NDAExemption[]>([]);
	let users = $state<User[]>([]);
	let loading = $state(true);
	let showCreate = $state(false);
	let showSignatures = $state(false);
	let showExemptions = $state(false);
	let showGrantForm = $state(false);
	let newName = $state('');
	let newContent = $state('');
	let message = $state('');
	let exemptUserId = $state('');
	let exemptReason = $state('');
	let exemptFile = $state<FileList | null>(null);
	let granting = $state(false);

	async function loadTemplates() {
		loading = true;
		try {
			const res = await api.get<NDATemplate[]>('/nda/templates');
			templates = res.data ?? [];
		} catch {
			templates = [];
		}
		loading = false;
	}

	async function loadSignatures() {
		try {
			const res = await api.get<NDASignature[]>('/nda/signatures');
			signatures = res.data ?? [];
		} catch {
			signatures = [];
		}
	}

	async function createTemplate() {
		if (!newName.trim() || !newContent.trim()) return;
		message = '';
		try {
			await api.post('/nda/templates', { name: newName, content: newContent });
			newName = '';
			newContent = '';
			showCreate = false;
			message = 'NDA template created.';
			await loadTemplates();
		} catch {
			message = 'Failed to create template.';
		}
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleDateString('en-US', {
			month: 'short', day: 'numeric', year: 'numeric'
		});
	}

	function toggleSignatures() {
		showSignatures = !showSignatures;
		if (showSignatures) loadSignatures();
	}

	async function loadExemptions() {
		try {
			const res = await api.get<NDAExemption[]>('/nda/exemptions');
			exemptions = res.data ?? [];
		} catch {
			exemptions = [];
		}
	}

	async function loadUsers() {
		try {
			const res = await api.get<User[]>('/users');
			users = res.data ?? [];
		} catch {
			users = [];
		}
	}

	function toggleExemptions() {
		showExemptions = !showExemptions;
		if (showExemptions) {
			loadExemptions();
			loadUsers();
		}
	}

	async function grantExemption() {
		if (!exemptUserId) return;
		message = '';
		granting = true;
		try {
			const form = new FormData();
			form.append('user_id', exemptUserId);
			if (exemptReason.trim()) form.append('reason', exemptReason.trim());
			if (exemptFile && exemptFile.length > 0) form.append('file', exemptFile[0]);
			await api.upload('/nda/exemptions', form);
			exemptUserId = '';
			exemptReason = '';
			exemptFile = null;
			showGrantForm = false;
			message = 'NDA exemption granted.';
			await loadExemptions();
		} catch {
			message = 'Failed to grant exemption.';
		}
		granting = false;
	}

	async function revokeExemption(ex: NDAExemption) {
		message = '';
		try {
			await api.delete(`/nda/exemptions/${ex.user_id}`);
			message = 'NDA exemption revoked.';
			await loadExemptions();
		} catch {
			message = 'Failed to revoke exemption.';
		}
	}

	async function downloadExemptionDocument(ex: NDAExemption) {
		try {
			await api.download(`/nda/exemptions/${ex.user_id}/document`, ex.file_name ?? 'nda.pdf');
		} catch {
			message = 'Failed to download document.';
		}
	}

	$effect(() => { loadTemplates(); });
</script>

<h1>NDA Templates</h1>

<div class="actions">
	<button class="btn-primary" onclick={() => (showCreate = !showCreate)}>
		{showCreate ? 'Cancel' : 'Create Template'}
	</button>
	<button class="btn-secondary" onclick={toggleSignatures}>
		{showSignatures ? 'Hide Signatures' : 'View Signatures'}
	</button>
	<button class="btn-secondary" onclick={toggleExemptions}>
		{showExemptions ? 'Hide Exemptions' : 'Manage Exemptions'}
	</button>
</div>

{#if message}
	<div class="message">{message}</div>
{/if}

{#if showCreate}
	<div class="create-form">
		<input type="text" placeholder="Template name" bind:value={newName} />
		<textarea rows="8" placeholder="NDA content (supports Markdown)" bind:value={newContent}></textarea>
		<button class="btn-primary" onclick={createTemplate}>Create Template</button>
	</div>
{/if}

{#if loading}
	<p class="loading">Loading templates...</p>
{:else if templates.length === 0}
	<p class="empty">No NDA templates. Create one to require investor signatures.</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>Name</th>
				<th>Version</th>
				<th>Status</th>
				<th>Created</th>
			</tr>
		</thead>
		<tbody>
			{#each templates as tmpl}
				<tr>
					<td>{tmpl.name}</td>
					<td>v{tmpl.version}</td>
					<td>
						<span class="status" class:active={tmpl.is_active} class:inactive={!tmpl.is_active}>
							{tmpl.is_active ? 'Active' : 'Inactive'}
						</span>
					</td>
					<td>{formatDate(tmpl.created_at)}</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}

{#if showSignatures}
	<h2>Signatures</h2>
	{#if signatures.length === 0}
		<p class="empty">No signatures yet.</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>Signer</th>
					<th>Email</th>
					<th>Company</th>
					<th>IP Address</th>
					<th>Signed</th>
				</tr>
			</thead>
			<tbody>
				{#each signatures as sig}
					<tr>
						<td>{sig.signer_name}</td>
						<td>{sig.signer_email}</td>
						<td>{sig.signer_company ?? ''}</td>
						<td class="mono">{sig.ip_address}</td>
						<td>{formatDate(sig.signed_at)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
{/if}

{#if showExemptions}
	<h2>Exemptions</h2>
	<p class="hint">
		Exempt users bypass the click-through NDA requirement. Optionally attach an
		externally executed NDA (e.g. a countersigned PDF) as evidence.
	</p>

	<div class="actions">
		<button class="btn-primary" onclick={() => (showGrantForm = !showGrantForm)}>
			{showGrantForm ? 'Cancel' : 'Grant Exemption'}
		</button>
	</div>

	{#if showGrantForm}
		<div class="create-form">
			<label>
				User
				<select bind:value={exemptUserId}>
					<option value="" disabled>Select a user…</option>
					{#each users as user}
						<option value={user.id}>{user.name} ({user.email}) — {user.role}</option>
					{/each}
				</select>
			</label>
			<label>
				Reason (optional)
				<input type="text" placeholder="e.g. NDA executed externally on 2026-07-01" bind:value={exemptReason} />
			</label>
			<label>
				Executed NDA document (optional)
				<input type="file" bind:files={exemptFile} />
			</label>
			<button class="btn-primary" onclick={grantExemption} disabled={!exemptUserId || granting}>
				{granting ? 'Granting…' : 'Grant Exemption'}
			</button>
		</div>
	{/if}

	{#if exemptions.length === 0}
		<p class="empty">No exemptions granted.</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>User</th>
					<th>Email</th>
					<th>Reason</th>
					<th>Document</th>
					<th>Granted By</th>
					<th>Granted</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				{#each exemptions as ex}
					<tr>
						<td>{ex.user_name ?? ex.user_id}</td>
						<td>{ex.user_email ?? ''}</td>
						<td>{ex.reason ?? ''}</td>
						<td>
							{#if ex.has_document}
								<button class="btn-link" onclick={() => downloadExemptionDocument(ex)}>
									{ex.file_name ?? 'Download'}
								</button>
							{:else}
								<span class="muted">None</span>
							{/if}
						</td>
						<td>{ex.granted_by_name ?? ex.granted_by}</td>
						<td>{formatDate(ex.created_at)}</td>
						<td>
							<button class="btn-danger" onclick={() => revokeExemption(ex)}>Revoke</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
{/if}

<style>
	h1 { font-weight: 400; font-size: 1.5rem; margin-bottom: 1.5rem; }
	h2 { font-weight: 400; font-size: 1.125rem; margin-top: 2rem; margin-bottom: 1rem; color: var(--dd-text-secondary); }

	.actions { display: flex; gap: 0.5rem; margin-bottom: 1rem; }

	.btn-primary {
		padding: 0.5rem 1rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		cursor: pointer;
		font-size: 0.8125rem;
	}

	.btn-secondary {
		padding: 0.5rem 1rem;
		background: var(--dd-surface);
		color: var(--dd-text);
		border: 1px solid var(--dd-border);
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

	.create-form {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		margin-bottom: 1.5rem;
		padding: 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
	}

	.create-form input, .create-form textarea {
		padding: 0.5rem 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.8125rem;
		width: 100%;
		resize: vertical;
	}

	.create-form textarea { font-family: monospace; }

	.create-form label {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		font-size: 0.75rem;
		color: var(--dd-text-secondary);
	}

	.create-form select {
		padding: 0.5rem 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.8125rem;
	}

	.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

	.btn-link {
		background: none;
		border: none;
		padding: 0;
		color: var(--dd-primary);
		cursor: pointer;
		font-size: 0.8125rem;
		text-decoration: underline;
	}

	.btn-danger {
		padding: 0.25rem 0.75rem;
		background: transparent;
		color: var(--dd-danger, #da1e28);
		border: 1px solid var(--dd-danger, #da1e28);
		cursor: pointer;
		font-size: 0.75rem;
	}

	.hint { font-size: 0.8125rem; color: var(--dd-text-secondary); margin-bottom: 1rem; }
	.muted { color: var(--dd-text-secondary); }

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

	.status.active { color: var(--dd-success, #24a148); }
	.status.inactive { color: var(--dd-text-secondary); }
	.mono { font-family: monospace; font-size: 0.75rem; }
	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }
</style>
