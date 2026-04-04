<script lang="ts">
	import type { AuditEntry } from '$types/api';
	import { api } from '$api/client';

	let entries = $state<AuditEntry[]>([]);
	let loading = $state(true);
	let filterAction = $state('');

	async function loadAudit() {
		loading = true;
		try {
			const params = filterAction ? `?action=${filterAction}` : '';
			const res = await api.get<AuditEntry[]>(`/audit${params}`);
			entries = res.data ?? [];
		} catch {
			entries = [];
		}
		loading = false;
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleString('en-US', {
			month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
		});
	}

	$effect(() => { loadAudit(); });
</script>

<h1>Audit Log</h1>

<div class="filters">
	<select bind:value={filterAction} onchange={loadAudit}>
		<option value="">All Actions</option>
		<option value="user.login">User Login</option>
		<option value="document.uploaded">Document Upload</option>
		<option value="document.downloaded">Document Download</option>
		<option value="document.viewed">Document View</option>
		<option value="permission.granted">Permission Granted</option>
		<option value="permission.revoked">Permission Revoked</option>
		<option value="nda.signed">NDA Signed</option>
	</select>
</div>

{#if loading}
	<p class="loading">Loading audit log...</p>
{:else if entries.length === 0}
	<p class="empty">No audit entries found.</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>Time</th>
				<th>User</th>
				<th>Action</th>
				<th>Resource</th>
				<th>IP</th>
			</tr>
		</thead>
		<tbody>
			{#each entries as entry}
				<tr>
					<td>{formatDate(entry.created_at)}</td>
					<td>{entry.user_email}</td>
					<td><span class="action-badge">{entry.action}</span></td>
					<td>{entry.resource_name ?? entry.resource_id ?? ''}</td>
					<td class="mono">{entry.ip_address ?? ''}</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}

<style>
	h1 { font-weight: 400; font-size: 1.5rem; margin-bottom: 1.5rem; }

	.filters { margin-bottom: 1rem; }

	select {
		padding: 0.5rem 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
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

	.action-badge {
		padding: 0.125rem 0.5rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		border-radius: 2px;
		font-size: 0.75rem;
		font-family: monospace;
	}

	.mono { font-family: monospace; font-size: 0.75rem; }
	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }
</style>
