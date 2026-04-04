<script lang="ts">
	import { api } from '$api/client';

	interface EngagementSummary {
		total_documents: number;
		total_views: number;
		total_downloads: number;
		unique_viewers: number;
		active_investors: number;
		open_questions: number;
		pending_ndas: number;
		recent_view_count: number;
	}

	let summary = $state<EngagementSummary | null>(null);
	let loading = $state(true);

	async function loadDashboard() {
		try {
			const res = await api.get<EngagementSummary>('/analytics/dashboard');
			summary = res.data ?? null;
		} catch {
			summary = null;
		}
		loading = false;
	}

	$effect(() => { loadDashboard(); });
</script>

<div class="page">
	<h1>Analytics Dashboard</h1>

	{#if loading}
		<p class="loading">Loading analytics...</p>
	{:else if !summary}
		<p class="empty">Unable to load analytics data.</p>
	{:else}
		<div class="metrics-grid">
			<div class="metric-card">
				<span class="metric-value">{summary.total_documents}</span>
				<span class="metric-label">Documents</span>
			</div>
			<div class="metric-card">
				<span class="metric-value">{summary.total_views}</span>
				<span class="metric-label">Total Views</span>
			</div>
			<div class="metric-card">
				<span class="metric-value">{summary.unique_viewers}</span>
				<span class="metric-label">Unique Viewers</span>
			</div>
			<div class="metric-card">
				<span class="metric-value">{summary.active_investors}</span>
				<span class="metric-label">Active Investors</span>
			</div>
			<div class="metric-card">
				<span class="metric-value">{summary.total_downloads}</span>
				<span class="metric-label">Downloads</span>
			</div>
			<div class="metric-card">
				<span class="metric-value">{summary.open_questions}</span>
				<span class="metric-label">Open Questions</span>
			</div>
			<div class="metric-card">
				<span class="metric-value">{summary.recent_view_count}</span>
				<span class="metric-label">Recent Views (7d)</span>
			</div>
			<div class="metric-card">
				<span class="metric-value">{summary.pending_ndas}</span>
				<span class="metric-label">Pending NDAs</span>
			</div>
		</div>
	{/if}
</div>

<style>
	.page { max-width: 1000px; margin: 0 auto; padding: 2rem; }
	h1 { font-weight: 400; font-size: 1.75rem; margin-bottom: 2rem; }

	.metrics-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
		gap: 1rem;
	}

	.metric-card {
		background: var(--dd-surface);
		border: 1px solid var(--dd-border);
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.metric-value {
		font-size: 2.5rem;
		font-weight: 300;
		color: var(--dd-primary);
	}

	.metric-label {
		font-size: 0.8125rem;
		color: var(--dd-text-secondary);
		margin-top: 0.5rem;
	}

	.loading, .empty { color: var(--dd-text-secondary); text-align: center; padding: 3rem; }
</style>
