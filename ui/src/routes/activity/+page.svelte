<script lang="ts">
	import { onMount } from 'svelte';
	import type { AuditEntry } from '$types/api';
	import { api } from '$api/client';

	const PAGE_SIZE = 50;

	let entries = $state<AuditEntry[]>([]);
	let loading = $state(true);
	let loadError = $state('');
	let filterAction = $state('');
	let page = $state(1);
	let total = $state(0);

	let totalPages = $derived(Math.max(1, Math.ceil(total / PAGE_SIZE)));

	async function loadAudit() {
		loading = true;
		loadError = '';
		try {
			const qs = new URLSearchParams();
			if (filterAction) qs.set('action', filterAction);
			qs.set('limit', String(PAGE_SIZE));
			qs.set('offset', String((page - 1) * PAGE_SIZE));
			const res = await api.get<AuditEntry[]>(`/audit?${qs.toString()}`);
			entries = res.data ?? [];
			total = res.meta?.total ?? entries.length;
		} catch {
			loadError = 'Failed to load the activity log.';
			entries = [];
			total = 0;
		}
		loading = false;
	}

	function onFilterChange() {
		page = 1;
		loadAudit();
	}

	function goToPage(p: number) {
		if (p < 1 || p > totalPages) return;
		page = p;
		loadAudit();
	}

	async function exportCsv() {
		const qs = filterAction ? `?action=${encodeURIComponent(filterAction)}` : '';
		try {
			await api.download(`/audit/export${qs}`, 'activity-log.csv');
		} catch {
			loadError = 'Export failed.';
		}
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleString('en-US', {
			month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit'
		});
	}

	onMount(() => { loadAudit(); });
</script>

<div class="page">
	<header class="page-header">
		<h1>Activity Log</h1>
		<button class="btn-secondary" onclick={exportCsv}>Export CSV</button>
	</header>

	<div class="filters">
		<select bind:value={filterAction} onchange={onFilterChange}>
			<option value="">All Actions</option>
			<option value="user.login">User Login</option>
			<option value="user.created">User Created</option>
			<option value="document.uploaded">Document Upload</option>
			<option value="document.downloaded">Document Download</option>
			<option value="document.viewed">Document View</option>
			<option value="document.new_version">New Version</option>
			<option value="permission.granted">Permission Granted</option>
			<option value="permission.revoked">Permission Revoked</option>
			<option value="qa.message_posted">Q&amp;A Message</option>
			<option value="nda.signed">NDA Signed</option>
		</select>
	</div>

	{#if loading}
		<p class="loading">Loading activity...</p>
	{:else if loadError}
		<p class="empty">{loadError} <button class="link" onclick={loadAudit}>Retry</button></p>
	{:else if entries.length === 0}
		<p class="empty">No activity recorded.</p>
	{:else}
		<table>
			<thead>
				<tr><th>Time</th><th>User</th><th>Action</th><th>Resource</th><th>Details</th><th>IP</th></tr>
			</thead>
			<tbody>
				{#each entries as entry}
					<tr>
						<td>{formatDate(entry.created_at)}</td>
						<td>{entry.user_email}</td>
						<td><span class="action-badge">{entry.action}</span></td>
						<td>{entry.resource_name ?? entry.resource_id ?? ''}</td>
						<td class="details">{entry.details ?? ''}</td>
						<td class="mono">{entry.ip_address ?? ''}</td>
					</tr>
				{/each}
			</tbody>
		</table>

		{#if totalPages > 1}
			<div class="pagination">
				<button class="page-btn" onclick={() => goToPage(page - 1)} disabled={page <= 1}>Previous</button>
				<span class="page-info">Page {page} of {totalPages} · {total} entries</span>
				<button class="page-btn" onclick={() => goToPage(page + 1)} disabled={page >= totalPages}>Next</button>
			</div>
		{/if}
	{/if}
</div>

<style>
	.page { max-width: 1100px; margin: 0 auto; padding: 2rem; }
	.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
	h1 { font-weight: 400; font-size: 1.75rem; }
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
	td { padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--dd-border); font-size: 0.8125rem; }
	.action-badge {
		padding: 0.125rem 0.5rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		border-radius: 2px;
		font-size: 0.75rem;
		font-family: monospace;
	}
	.details { color: var(--dd-text-secondary); font-size: 0.75rem; }
	.mono { font-family: monospace; font-size: 0.75rem; }
	.btn-secondary {
		padding: 0.5rem 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		cursor: pointer;
		font-size: 0.8125rem;
	}
	.pagination { display: flex; align-items: center; justify-content: center; gap: 1rem; margin-top: 1.5rem; }
	.page-btn {
		padding: 0.5rem 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		cursor: pointer;
		font-size: 0.8125rem;
	}
	.page-btn:disabled { opacity: 0.4; cursor: default; }
	.page-info { font-size: 0.8125rem; color: var(--dd-text-secondary); }
	.link { background: none; border: none; color: var(--dd-primary); cursor: pointer; text-decoration: underline; font-size: inherit; }
	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }
</style>
