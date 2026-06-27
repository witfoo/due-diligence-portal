<script lang="ts">
	import type { Category } from '$types/api';
	import { api, ApiError } from '$api/client';
	import { goto } from '$app/navigation';

	let categories = $state<Category[]>([]);
	let name = $state('');
	let description = $state('');
	let categoryId = $state('');
	let tags = $state('');
	let file = $state<File | null>(null);
	let submitting = $state(false);
	let error = $state('');

	async function loadCategories() {
		try {
			const res = await api.get<Category[]>('/categories');
			categories = flatten(res.data ?? []);
			if (categories.length > 0 && !categoryId) categoryId = categories[0].id;
		} catch {
			error = 'Failed to load categories.';
		}
	}

	// The categories endpoint returns a nested tree; flatten it for the dropdown.
	function flatten(tree: Category[], depth = 0): Category[] {
		const out: Category[] = [];
		for (const c of tree) {
			out.push({ ...c, name: `${'— '.repeat(depth)}${c.name}` });
			if (c.children?.length) out.push(...flatten(c.children, depth + 1));
		}
		return out;
	}

	function onFileChange(e: Event) {
		const target = e.target as HTMLInputElement;
		file = target.files?.[0] ?? null;
		if (file && !name) name = file.name;
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		if (!file) {
			error = 'Please choose a file.';
			return;
		}
		if (!name.trim()) {
			error = 'Name is required.';
			return;
		}
		if (!categoryId) {
			error = 'Category is required.';
			return;
		}

		submitting = true;
		try {
			const form = new FormData();
			form.append('file', file);
			form.append('name', name.trim());
			form.append('description', description.trim());
			form.append('category_id', categoryId);
			form.append('tags', tags.trim());
			await api.upload('/documents', form);
			await goto('/documents');
		} catch (err) {
			if (err instanceof ApiError && err.status === 413) {
				error = 'File exceeds the maximum upload size.';
			} else {
				error = 'Upload failed. Please try again.';
			}
		} finally {
			submitting = false;
		}
	}

	$effect(() => {
		loadCategories();
	});
</script>

<div class="page">
	<header class="page-header">
		<h1>Upload Document</h1>
		<a href="/documents" class="btn-secondary">Cancel</a>
	</header>

	{#if error}
		<div class="error-msg">{error}</div>
	{/if}

	<form onsubmit={handleSubmit}>
		<div class="field">
			<label for="file">File</label>
			<input id="file" type="file" onchange={onFileChange} required />
		</div>
		<div class="field">
			<label for="name">Name</label>
			<input id="name" type="text" bind:value={name} placeholder="Document name" required />
		</div>
		<div class="field">
			<label for="description">Description</label>
			<textarea id="description" bind:value={description} rows="3" placeholder="Optional description"></textarea>
		</div>
		<div class="field">
			<label for="category">Category</label>
			<select id="category" bind:value={categoryId} required>
				{#each categories as cat}
					<option value={cat.id}>{cat.name}</option>
				{/each}
			</select>
		</div>
		<div class="field">
			<label for="tags">Tags</label>
			<input id="tags" type="text" bind:value={tags} placeholder="Comma-separated tags (optional)" />
		</div>
		<button type="submit" class="btn-primary" disabled={submitting}>
			{submitting ? 'Uploading…' : 'Upload'}
		</button>
	</form>
</div>

<style>
	.page { max-width: 640px; margin: 0 auto; padding: 2rem; }
	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.5rem;
	}
	h1 { font-weight: 400; font-size: 1.75rem; }
	.error-msg {
		padding: 0.75rem 1rem;
		background: #da1e2815;
		border: 1px solid var(--dd-error, #da1e28);
		color: var(--dd-error, #da1e28);
		font-size: 0.875rem;
		margin-bottom: 1.5rem;
	}
	.field { margin-bottom: 1.25rem; }
	label {
		display: block;
		font-size: 0.875rem;
		color: var(--dd-text-secondary);
		margin-bottom: 0.5rem;
	}
	input, textarea, select {
		width: 100%;
		padding: 0.625rem 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.875rem;
	}
	.btn-primary {
		padding: 0.75rem 1.5rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		cursor: pointer;
		font-size: 0.875rem;
	}
	.btn-primary:disabled { opacity: 0.5; cursor: wait; }
	.btn-secondary {
		padding: 0.5rem 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		text-decoration: none;
		font-size: 0.875rem;
	}
</style>
