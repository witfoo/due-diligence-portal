<script lang="ts">
	import type { QAThread, QAMessage } from '$types/api';
	import { api } from '$api/client';
	import { page } from '$app/state';
	import { authStore } from '$stores/authStore.svelte';

	let thread = $state<QAThread | null>(null);
	let messages = $state<QAMessage[]>([]);
	let loading = $state(true);
	let error = $state('');
	let reply = $state('');
	let postInternal = $state(false);
	let posting = $state(false);

	let threadId = $derived(page.params.threadId ?? '');
	let isStaff = $derived(authStore.isAdmin || authStore.isCompanyMember);

	async function load(id: string) {
		loading = true;
		error = '';
		try {
			const res = await api.get<{ thread: QAThread; messages: QAMessage[] }>(`/qa/${id}`);
			thread = res.data?.thread ?? null;
			messages = res.data?.messages ?? [];
		} catch {
			error = 'Thread not found or you do not have access.';
			thread = null;
		}
		loading = false;
	}

	async function postReply(e: Event) {
		e.preventDefault();
		if (!reply.trim()) return;
		posting = true;
		try {
			await api.post(`/qa/${threadId}/messages`, {
				body: reply.trim(),
				...(isStaff ? { is_internal: postInternal } : {})
			});
			reply = '';
			postInternal = false;
			await load(threadId);
		} catch {
			error = 'Failed to post reply.';
		} finally {
			posting = false;
		}
	}

	async function changeStatus(status: string) {
		try {
			await api.patch(`/qa/${threadId}/status`, { status });
			await load(threadId);
		} catch {
			error = 'Failed to update status.';
		}
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleString('en-US', {
			month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit'
		});
	}

	$effect(() => {
		if (threadId) load(threadId);
	});
</script>

<div class="page">
	<a href="/qa" class="back">&larr; Back to Q&amp;A</a>

	{#if loading}
		<p class="loading">Loading…</p>
	{:else if error && !thread}
		<p class="empty">{error}</p>
	{:else if thread}
		<header class="thread-header">
			<h1>{thread.subject}</h1>
			<span class="status status-{thread.status}">{thread.status}</span>
		</header>

		{#if isStaff}
			<div class="status-actions">
				<span>Set status:</span>
				<button onclick={() => changeStatus('open')} disabled={thread.status === 'open'}>Open</button>
				<button onclick={() => changeStatus('answered')} disabled={thread.status === 'answered'}>Answered</button>
				<button onclick={() => changeStatus('closed')} disabled={thread.status === 'closed'}>Closed</button>
			</div>
		{/if}

		{#if error}<div class="error-msg">{error}</div>{/if}

		<div class="messages">
			{#if messages.length === 0}
				<p class="empty">No messages yet.</p>
			{:else}
				{#each messages as msg}
					<div class="message" class:internal={msg.is_internal}>
						<div class="message-head">
							<span class="author">{msg.author_name ?? msg.author_email ?? 'User'}</span>
							{#if msg.is_internal}<span class="badge">Internal</span>{/if}
							<span class="date">{formatDate(msg.created_at)}</span>
						</div>
						<p class="body">{msg.body}</p>
					</div>
				{/each}
			{/if}
		</div>

		<form class="reply-form" onsubmit={postReply}>
			<label for="reply">Reply</label>
			<textarea id="reply" bind:value={reply} rows="3" placeholder="Write a reply…" required></textarea>
			<div class="reply-actions">
				{#if isStaff}
					<label class="internal-toggle">
						<input type="checkbox" bind:checked={postInternal} />
						Internal note (not visible to investors)
					</label>
				{/if}
				<button type="submit" class="btn-primary" disabled={posting}>
					{posting ? 'Posting…' : 'Post Reply'}
				</button>
			</div>
		</form>
	{/if}
</div>

<style>
	.page { max-width: 800px; margin: 0 auto; padding: 2rem; }
	.back { color: var(--dd-primary); text-decoration: none; font-size: 0.875rem; }
	.thread-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin: 1rem 0;
		gap: 1rem;
	}
	h1 { font-weight: 400; font-size: 1.5rem; }
	.status { text-transform: uppercase; font-size: 0.75rem; font-weight: 600; }
	.status-open { color: var(--dd-warning); }
	.status-answered { color: var(--dd-success); }
	.status-closed { color: var(--dd-text-secondary); }
	.status-actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
		font-size: 0.8125rem;
		color: var(--dd-text-secondary);
	}
	.status-actions button {
		padding: 0.25rem 0.75rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		cursor: pointer;
		font-size: 0.75rem;
	}
	.status-actions button:disabled { opacity: 0.4; cursor: default; }
	.error-msg {
		padding: 0.5rem 1rem;
		background: #da1e2815;
		border: 1px solid var(--dd-error, #da1e28);
		color: var(--dd-error, #da1e28);
		font-size: 0.875rem;
		margin-bottom: 1rem;
	}
	.messages { display: flex; flex-direction: column; gap: 0.75rem; margin-bottom: 2rem; }
	.message {
		padding: 1rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
	}
	.message.internal { border-left: 3px solid var(--dd-warning); }
	.message-head {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 0.5rem;
		font-size: 0.75rem;
	}
	.author { font-weight: 600; color: var(--dd-text); }
	.badge {
		background: var(--dd-warning);
		color: #000;
		padding: 0.0625rem 0.375rem;
		border-radius: 2px;
		font-size: 0.625rem;
		text-transform: uppercase;
	}
	.date { color: var(--dd-text-secondary); margin-left: auto; }
	.body { margin: 0; font-size: 0.875rem; white-space: pre-wrap; }
	.reply-form { display: flex; flex-direction: column; gap: 0.75rem; }
	.reply-form label { font-size: 0.875rem; color: var(--dd-text-secondary); }
	textarea {
		padding: 0.625rem 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.875rem;
		font-family: inherit;
	}
	.reply-actions {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
	}
	.internal-toggle {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.8125rem;
		color: var(--dd-text-secondary);
	}
	.btn-primary {
		padding: 0.625rem 1.25rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		cursor: pointer;
		font-size: 0.875rem;
		margin-left: auto;
	}
	.btn-primary:disabled { opacity: 0.5; cursor: wait; }
	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 2rem; }
</style>
