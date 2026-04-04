<script lang="ts">
	import type { Category } from '$types/api';
	import { api } from '$api/client';

	let categories = $state<Category[]>([]);
	let loading = $state(true);
	let showAdd = $state(false);
	let newName = $state('');
	let newSlug = $state('');
	let newDescription = $state('');
	let message = $state('');

	async function loadCategories() {
		loading = true;
		try {
			const res = await api.get<Category[]>('/categories');
			categories = res.data ?? [];
		} catch {
			categories = [];
		}
		loading = false;
	}

	function autoSlug() {
		newSlug = newName.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
	}

	async function addCategory() {
		if (!newName.trim()) return;
		message = '';
		try {
			await api.post('/categories', {
				name: newName,
				slug: newSlug || newName.toLowerCase().replace(/[^a-z0-9]+/g, '-'),
				description: newDescription
			});
			newName = '';
			newSlug = '';
			newDescription = '';
			showAdd = false;
			message = 'Category created.';
			await loadCategories();
		} catch {
			message = 'Failed to create category.';
		}
	}

	$effect(() => { loadCategories(); });
</script>

<h1>Categories</h1>

<div class="actions">
	<button class="btn-primary" onclick={() => (showAdd = !showAdd)}>
		{showAdd ? 'Cancel' : 'Add Category'}
	</button>
</div>

{#if message}
	<div class="message">{message}</div>
{/if}

{#if showAdd}
	<div class="add-form">
		<div class="form-row">
			<input type="text" placeholder="Category name" bind:value={newName} oninput={autoSlug} />
			<input type="text" placeholder="slug (auto)" bind:value={newSlug} />
		</div>
		<input type="text" placeholder="Description" bind:value={newDescription} class="full-width" />
		<button class="btn-primary" onclick={addCategory}>Create</button>
	</div>
{/if}

{#if loading}
	<p class="loading">Loading categories...</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>Name</th>
				<th>Slug</th>
				<th>Description</th>
				<th>Order</th>
			</tr>
		</thead>
		<tbody>
			{#each categories as cat}
				<tr>
					<td>{cat.name}</td>
					<td class="mono">{cat.slug}</td>
					<td class="desc">{cat.description ?? ''}</td>
					<td>{cat.sort_order}</td>
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

	.add-form {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
		padding: 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
	}

	.form-row { display: flex; gap: 0.5rem; }

	.add-form input {
		padding: 0.5rem 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.8125rem;
		flex: 1;
	}

	.full-width { width: 100%; }

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

	.mono { font-family: monospace; font-size: 0.75rem; }
	.desc { color: var(--dd-text-secondary); max-width: 300px; }
	.loading { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }
</style>
