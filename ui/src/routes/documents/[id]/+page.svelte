<script lang="ts">
	import type { Document, DocumentVersion } from '$types/api';
	import { api } from '$api/client';
	import { page } from '$app/state';

	let doc = $state<Document | null>(null);
	let versions = $state<DocumentVersion[]>([]);
	let loading = $state(true);
	let error = $state('');
	let downloadError = $state('');

	let docId = $derived(page.params.id ?? '');

	async function load(id: string) {
		loading = true;
		error = '';
		try {
			const res = await api.get<{ document: Document; versions: DocumentVersion[] }>(`/documents/${id}`);
			doc = res.data?.document ?? null;
			versions = res.data?.versions ?? [];
			// Record a view event (best-effort; failure must not block the page).
			api.post('/analytics/view-event', { document_id: id }).catch(() => {});
		} catch {
			error = 'Document not found or you do not have access.';
			doc = null;
		}
		loading = false;
	}

	async function downloadCurrent() {
		downloadError = '';
		try {
			await api.download(`/documents/${docId}/download`, doc?.name ?? 'document');
		} catch {
			downloadError = 'Download failed.';
		}
	}

	async function downloadVersion(v: number) {
		downloadError = '';
		try {
			await api.download(`/documents/${docId}/versions/${v}`, `${doc?.name ?? 'document'}-v${v}`);
		} catch {
			downloadError = 'Download failed.';
		}
	}

	function formatSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleString('en-US', {
			year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
		});
	}

	$effect(() => {
		if (docId) load(docId);
	});
</script>

<div class="page">
	<a href="/documents" class="back">&larr; Back to Documents</a>

	{#if loading}
		<p class="loading">Loading…</p>
	{:else if error}
		<p class="empty">{error}</p>
	{:else if doc}
		<header class="doc-header">
			<h1>{doc.name}</h1>
			<button class="btn-primary" onclick={downloadCurrent}>Download</button>
		</header>

		{#if downloadError}<div class="error-msg">{downloadError}</div>{/if}
		{#if doc.description}<p class="description">{doc.description}</p>{/if}

		<dl class="meta">
			<div><dt>Category</dt><dd>{doc.category_name ?? '—'}</dd></div>
			<div><dt>Current version</dt><dd>v{doc.current_version}</dd></div>
			<div><dt>Size</dt><dd>{formatSize(doc.file_size)}</dd></div>
			<div><dt>Type</dt><dd>{doc.mime_type}</dd></div>
			<div><dt>Uploaded by</dt><dd>{doc.uploader_name ?? '—'}</dd></div>
			<div><dt>Uploaded</dt><dd>{formatDate(doc.created_at)}</dd></div>
			{#if doc.tags}<div><dt>Tags</dt><dd>{doc.tags}</dd></div>{/if}
		</dl>

		<h2>Version history</h2>
		{#if versions.length === 0}
			<p class="empty">No versions recorded.</p>
		{:else}
			<table>
				<thead>
					<tr><th>Version</th><th>Size</th><th>Change note</th><th>Date</th><th></th></tr>
				</thead>
				<tbody>
					{#each versions as v}
						<tr>
							<td>v{v.version_number}</td>
							<td>{formatSize(v.file_size)}</td>
							<td>{v.change_note ?? '—'}</td>
							<td>{formatDate(v.created_at)}</td>
							<td><button class="btn-small" onclick={() => downloadVersion(v.version_number)}>Download</button></td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	{/if}
</div>

<style>
	.page { max-width: 900px; margin: 0 auto; padding: 2rem; }
	.back { color: var(--dd-primary); text-decoration: none; font-size: 0.875rem; }
	.doc-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin: 1rem 0;
	}
	h1 { font-weight: 400; font-size: 1.75rem; }
	h2 { font-weight: 400; font-size: 1.25rem; margin: 2rem 0 1rem; }
	.description { color: var(--dd-text-secondary); margin-bottom: 1.5rem; }
	.error-msg {
		padding: 0.5rem 1rem;
		background: #da1e2815;
		border: 1px solid var(--dd-error, #da1e28);
		color: var(--dd-error, #da1e28);
		font-size: 0.875rem;
		margin-bottom: 1rem;
	}
	.meta {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.75rem 2rem;
		margin: 0;
	}
	.meta dt { font-size: 0.75rem; text-transform: uppercase; color: var(--dd-text-secondary); }
	.meta dd { margin: 0.125rem 0 0; font-size: 0.875rem; }
	table { width: 100%; border-collapse: collapse; }
	th {
		text-align: left;
		padding: 0.75rem 1rem;
		background: var(--dd-surface);
		border-bottom: 1px solid var(--dd-border);
		font-size: 0.75rem;
		text-transform: uppercase;
		color: var(--dd-text-secondary);
	}
	td { padding: 0.75rem 1rem; border-bottom: 1px solid var(--dd-border); font-size: 0.875rem; }
	.btn-primary {
		padding: 0.625rem 1.25rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		cursor: pointer;
		font-size: 0.875rem;
	}
	.btn-small {
		padding: 0.25rem 0.75rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		cursor: pointer;
		font-size: 0.75rem;
	}
	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }
</style>
