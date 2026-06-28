<script lang="ts">
	import { onMount } from 'svelte';
	import type { Category, Document } from '$types/api';
	import { api } from '$api/client';

	const PAGE_SIZE = 50;

	let documents = $state<Document[]>([]);
	let categories = $state<Category[]>([]);
	let selectedCategory = $state('');
	let searchQuery = $state('');
	let searchMode = $state(false);
	let loading = $state(true);
	let loadError = $state('');
	let page = $state(1);
	let total = $state(0);

	let totalPages = $derived(Math.max(1, Math.ceil(total / PAGE_SIZE)));

	async function loadDocuments() {
		loading = true;
		loadError = '';
		searchMode = false;
		try {
			const qs = new URLSearchParams();
			if (selectedCategory) qs.set('category_id', selectedCategory);
			qs.set('limit', String(PAGE_SIZE));
			qs.set('offset', String((page - 1) * PAGE_SIZE));
			const res = await api.get<Document[]>(`/documents?${qs.toString()}`);
			documents = res.data ?? [];
			total = res.meta?.total ?? documents.length;
		} catch {
			loadError = 'Failed to load documents.';
			documents = [];
			total = 0;
		}
		loading = false;
	}

	// The categories endpoint returns a nested tree; flatten it and keep only
	// categories that actually contain documents (so empty categories are not
	// offered as filters that always return nothing).
	function flattenCategories(tree: Category[], depth = 0): Category[] {
		const out: Category[] = [];
		for (const c of tree) {
			out.push({ ...c, name: `${'— '.repeat(depth)}${c.name}` });
			if (c.children?.length) out.push(...flattenCategories(c.children, depth + 1));
		}
		return out;
	}

	async function loadCategories() {
		try {
			const res = await api.get<Category[]>('/categories');
			categories = flattenCategories(res.data ?? []).filter((c) => (c.document_count ?? 0) > 0);
		} catch {
			categories = [];
		}
	}

	function onCategoryChange() {
		page = 1;
		searchQuery = '';
		loadDocuments();
	}

	function goToPage(p: number) {
		if (p < 1 || p > totalPages) return;
		page = p;
		loadDocuments();
	}

	async function handleSearch() {
		if (!searchQuery.trim()) {
			page = 1;
			await loadDocuments();
			return;
		}
		loading = true;
		loadError = '';
		searchMode = true;
		try {
			const res = await api.post<Document[]>('/documents/search', { query: searchQuery });
			documents = res.data ?? [];
			total = documents.length;
		} catch {
			loadError = 'Search failed.';
			documents = [];
		}
		loading = false;
	}

	async function downloadDoc(id: string, name: string) {
		try {
			await api.download(`/documents/${id}/download`, name);
		} catch {
			// surfaced inline below via a transient flag is overkill here; ignore.
		}
	}

	function formatSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleDateString('en-US', {
			year: 'numeric', month: 'short', day: 'numeric'
		});
	}

	onMount(() => {
		loadCategories();
		loadDocuments();
	});
</script>

<div class="page">
	<header class="page-header">
		<h1>Documents</h1>
		<a href="/documents/upload" class="btn-primary">Upload Document</a>
	</header>

	<div class="filters">
		<div class="search-bar">
			<input
				type="text"
				placeholder="Search documents..."
				bind:value={searchQuery}
				onkeydown={(e) => e.key === 'Enter' && handleSearch()}
			/>
			<button onclick={handleSearch}>Search</button>
		</div>

		<select bind:value={selectedCategory} onchange={onCategoryChange}>
			<option value="">All Categories</option>
			{#each categories as cat}
				<option value={cat.id}>{cat.name} ({cat.document_count})</option>
			{/each}
		</select>
	</div>

	{#if loading}
		<div class="loading">Loading documents...</div>
	{:else if loadError}
		<div class="empty">{loadError} <button class="link" onclick={loadDocuments}>Retry</button></div>
	{:else if documents.length === 0}
		<div class="empty">No documents found.</div>
	{:else}
		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Category</th>
					<th>Size</th>
					<th>Version</th>
					<th>Uploaded</th>
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>
				{#each documents as doc}
					<tr>
						<td>
							<a href="/documents/{doc.id}">{doc.name}</a>
							{#if doc.description}
								<span class="description">{doc.description}</span>
							{/if}
						</td>
						<td>{doc.category_name ?? ''}</td>
						<td>{formatSize(doc.file_size)}</td>
						<td>v{doc.current_version}</td>
						<td>{formatDate(doc.created_at)}</td>
						<td>
							<button class="btn-small" onclick={() => downloadDoc(doc.id, doc.name)}>Download</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
			{#if !searchMode && totalPages > 1}
				<div class="pagination">
					<button class="page-btn" onclick={() => goToPage(page - 1)} disabled={page <= 1}>Previous</button>
					<span class="page-info">Page {page} of {totalPages} · {total} documents</span>
					<button class="page-btn" onclick={() => goToPage(page + 1)} disabled={page >= totalPages}>Next</button>
				</div>
			{/if}
	{/if}
</div>

<style>
	.pagination {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 1rem;
		margin-top: 1.5rem;
	}
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
	.page {
		max-width: 1200px;
		margin: 0 auto;
		padding: 2rem;
	}

	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 2rem;
	}

	h1 { font-weight: 400; font-size: 1.75rem; }

	.btn-primary {
		padding: 0.625rem 1.25rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		text-decoration: none;
		font-size: 0.875rem;
	}

	.filters {
		display: flex;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.search-bar {
		display: flex;
		flex: 1;
		gap: 0.5rem;
	}

	.search-bar input {
		flex: 1;
		padding: 0.625rem 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.875rem;
	}

	.search-bar button, select {
		padding: 0.625rem 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		cursor: pointer;
	}

	table {
		width: 100%;
		border-collapse: collapse;
	}

	th {
		text-align: left;
		padding: 0.75rem 1rem;
		background: var(--dd-surface);
		border-bottom: 1px solid var(--dd-border);
		font-size: 0.75rem;
		text-transform: uppercase;
		color: var(--dd-text-secondary);
	}

	td {
		padding: 0.75rem 1rem;
		border-bottom: 1px solid var(--dd-border);
		font-size: 0.875rem;
	}

	td a { color: var(--dd-primary); }

	.description {
		display: block;
		font-size: 0.75rem;
		color: var(--dd-text-secondary);
		margin-top: 0.25rem;
	}

	.btn-small {
		padding: 0.25rem 0.75rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		text-decoration: none;
		font-size: 0.75rem;
		cursor: pointer;
	}

	.link {
		background: none;
		border: none;
		color: var(--dd-primary);
		cursor: pointer;
		text-decoration: underline;
		font-size: inherit;
	}

	.loading, .empty {
		text-align: center;
		padding: 3rem;
		color: var(--dd-text-secondary);
	}
</style>
