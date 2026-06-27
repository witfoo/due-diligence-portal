<script lang="ts">
	import type { QAThread } from '$types/api';
	import { api } from '$api/client';

	let threads = $state<QAThread[]>([]);
	let loading = $state(true);
	let filterStatus = $state('');
	let showNewForm = $state(false);
	let newSubject = $state('');
	let loadError = $state('');
	let formError = $state('');

	async function loadThreads() {
		loading = true;
		loadError = '';
		try {
			const params = filterStatus ? `?status=${filterStatus}` : '';
			const res = await api.get<QAThread[]>(`/qa${params}`);
			threads = res.data ?? [];
		} catch {
			loadError = 'Failed to load questions.';
			threads = [];
		}
		loading = false;
	}

	async function createThread() {
		if (!newSubject.trim()) return;
		formError = '';
		try {
			await api.post('/qa', { subject: newSubject });
			newSubject = '';
			showNewForm = false;
			await loadThreads();
		} catch {
			formError = 'Failed to create question.';
		}
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleDateString('en-US', {
			month: 'short', day: 'numeric', year: 'numeric'
		});
	}

	function statusColor(status: string): string {
		switch (status) {
			case 'open': return 'var(--dd-warning)';
			case 'answered': return 'var(--dd-success)';
			case 'closed': return 'var(--dd-text-secondary)';
			default: return 'var(--dd-text)';
		}
	}

	$effect(() => { loadThreads(); });
</script>

<div class="page">
	<header class="page-header">
		<h1>Questions & Answers</h1>
		<button class="btn-primary" onclick={() => showNewForm = !showNewForm}>
			{showNewForm ? 'Cancel' : 'Ask Question'}
		</button>
	</header>

	{#if formError}<div class="error-msg">{formError}</div>{/if}

	{#if showNewForm}
		<div class="new-form">
			<input
				type="text"
				placeholder="What would you like to know?"
				bind:value={newSubject}
				onkeydown={(e) => e.key === 'Enter' && createThread()}
			/>
			<button onclick={createThread}>Submit</button>
		</div>
	{/if}

	<div class="filters">
		<select bind:value={filterStatus} onchange={loadThreads}>
			<option value="">All Status</option>
			<option value="open">Open</option>
			<option value="answered">Answered</option>
			<option value="closed">Closed</option>
		</select>
	</div>

	{#if loading}
		<p class="loading">Loading threads...</p>
	{:else if loadError}
		<p class="empty">{loadError} <button class="link" onclick={loadThreads}>Retry</button></p>
	{:else if threads.length === 0}
		<p class="empty">No questions yet.</p>
	{:else}
		<div class="thread-list">
			{#each threads as thread}
				<a href="/qa/{thread.id}" class="thread-card">
					<div class="thread-header">
						<span class="status" style="color: {statusColor(thread.status)}">{thread.status}</span>
						<span class="date">{formatDate(thread.created_at)}</span>
					</div>
					<h3>{thread.subject}</h3>
					<div class="thread-meta">
						<span>Asked by {thread.asked_by_name ?? 'Unknown'}</span>
						{#if thread.message_count}
							<span>{thread.message_count} messages</span>
						{/if}
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>

<style>
	.page { max-width: 800px; margin: 0 auto; padding: 2rem; }

	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.5rem;
	}

	h1 { font-weight: 400; font-size: 1.75rem; }

	.btn-primary {
		padding: 0.625rem 1.25rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		cursor: pointer;
		font-size: 0.875rem;
	}

	.new-form {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
		padding: 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
	}

	.new-form input {
		flex: 1;
		padding: 0.625rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
	}

	.new-form button {
		padding: 0.625rem 1rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		cursor: pointer;
	}

	.filters { margin-bottom: 1rem; }

	select {
		padding: 0.5rem 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
	}

	.thread-list { display: flex; flex-direction: column; gap: 0.5rem; }

	.thread-card {
		display: block;
		padding: 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		text-decoration: none;
		color: var(--dd-text);
	}

	.thread-card:hover { border-color: var(--dd-primary); }

	.thread-header {
		display: flex;
		justify-content: space-between;
		margin-bottom: 0.5rem;
		font-size: 0.75rem;
	}

	.status { text-transform: uppercase; font-weight: 600; }
	.date { color: var(--dd-text-secondary); }

	h3 { font-weight: 400; font-size: 1rem; margin: 0 0 0.5rem; }

	.thread-meta {
		display: flex;
		gap: 1rem;
		font-size: 0.75rem;
		color: var(--dd-text-secondary);
	}

	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }

	.error-msg {
		padding: 0.5rem 1rem;
		background: #da1e2815;
		border: 1px solid var(--dd-error, #da1e28);
		color: var(--dd-error, #da1e28);
		font-size: 0.875rem;
		margin-bottom: 1rem;
	}

	.link {
		background: none;
		border: none;
		color: var(--dd-primary);
		cursor: pointer;
		text-decoration: underline;
		font-size: inherit;
	}
</style>
