<script lang="ts">
	import type { NDATemplate } from '$types/api';
	import { api, ApiError } from '$api/client';
	import { goto } from '$app/navigation';

	let template = $state<NDATemplate | null>(null);
	let loading = $state(true);
	let signerCompany = $state('');
	let agreed = $state(false);
	let submitting = $state(false);
	let error = $state('');
	let alreadySigned = $state(false);

	async function load() {
		loading = true;
		error = '';
		try {
			// If already signed, send the user on to the data room.
			const status = await api.get<{ signed: boolean }>('/nda/status');
			if (status.data?.signed) {
				alreadySigned = true;
				loading = false;
				return;
			}
			const res = await api.get<NDATemplate>('/nda/active');
			template = res.data ?? null;
			if (!template) error = 'No NDA is currently required.';
		} catch (err) {
			if (err instanceof ApiError && err.status === 404) {
				// No active NDA required — nothing to sign.
				alreadySigned = true;
			} else {
				error = 'Failed to load the NDA.';
			}
		}
		loading = false;
	}

	async function sign(e: Event) {
		e.preventDefault();
		if (!template || !agreed) return;
		submitting = true;
		error = '';
		try {
			await api.post(`/nda/sign/${template.id}`, { signer_company: signerCompany.trim() });
			await goto('/documents');
		} catch (err) {
			if (err instanceof ApiError && err.status === 409) {
				await goto('/documents');
			} else {
				error = 'Failed to record your signature. Please try again.';
			}
		} finally {
			submitting = false;
		}
	}

	$effect(() => {
		load();
	});
</script>

<div class="page">
	<h1>Non-Disclosure Agreement</h1>

	{#if loading}
		<p class="loading">Loading…</p>
	{:else if alreadySigned}
		<p class="info">You have already accepted the NDA.</p>
		<a href="/documents" class="btn-primary">Continue to the data room</a>
	{:else if template}
		{#if error}<div class="error-msg">{error}</div>{/if}
		<h2>{template.name}</h2>
		<div class="nda-content">{template.content}</div>

		<form onsubmit={sign}>
			<div class="field">
				<label for="company">Your Company (optional)</label>
				<input id="company" type="text" bind:value={signerCompany} />
			</div>
			<label class="agree">
				<input type="checkbox" bind:checked={agreed} />
				I have read and agree to the terms of this Non-Disclosure Agreement.
			</label>
			<button type="submit" class="btn-primary" disabled={!agreed || submitting}>
				{submitting ? 'Submitting…' : 'Accept &amp; Continue'}
			</button>
		</form>
	{:else}
		<p class="info">{error || 'No NDA is currently required.'}</p>
		<a href="/documents" class="btn-primary">Continue</a>
	{/if}
</div>

<style>
	.page { max-width: 760px; margin: 0 auto; padding: 2rem; }
	h1 { font-weight: 400; font-size: 1.75rem; margin-bottom: 1.5rem; }
	h2 { font-weight: 400; font-size: 1.25rem; margin-bottom: 1rem; }
	.nda-content {
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		padding: 1.5rem;
		max-height: 50vh;
		overflow-y: auto;
		white-space: pre-wrap;
		font-size: 0.875rem;
		line-height: 1.6;
		margin-bottom: 1.5rem;
	}
	.error-msg {
		padding: 0.75rem 1rem;
		background: #da1e2815;
		border: 1px solid var(--dd-error, #da1e28);
		color: var(--dd-error, #da1e28);
		font-size: 0.875rem;
		margin-bottom: 1rem;
	}
	.field { margin-bottom: 1rem; max-width: 360px; }
	label { display: block; font-size: 0.875rem; color: var(--dd-text-secondary); margin-bottom: 0.5rem; }
	input[type='text'] {
		width: 100%;
		padding: 0.625rem 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-size: 0.875rem;
	}
	.agree {
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		font-size: 0.875rem;
		color: var(--dd-text);
		margin-bottom: 1.5rem;
	}
	.agree input { margin-top: 0.2rem; }
	.btn-primary {
		display: inline-block;
		padding: 0.75rem 1.5rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		cursor: pointer;
		font-size: 0.875rem;
		text-decoration: none;
	}
	.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
	.loading, .info { color: var(--dd-text-secondary); padding: 1rem 0; }
</style>
