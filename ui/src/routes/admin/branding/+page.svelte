<script lang="ts">
	import type { BrandingConfig } from '$types/api';
	import { api } from '$api/client';
	import { applyBrandingCSS } from '$theme/branding-engine';

	let config = $state<BrandingConfig | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let message = $state('');

	async function loadConfig() {
		try {
			const res = await api.get<BrandingConfig>('/branding');
			config = res.data ?? null;
		} catch {
			config = null;
		}
		loading = false;
	}

	async function saveConfig() {
		if (!config) return;
		saving = true;
		message = '';
		try {
			await api.put('/branding', config);
			applyBrandingCSS(config);
			message = 'Branding saved successfully.';
		} catch {
			message = 'Failed to save branding.';
		}
		saving = false;
	}

	async function resetConfig() {
		try {
			await api.delete('/branding');
			await loadConfig();
			if (config) applyBrandingCSS(config);
			message = 'Branding reset to defaults.';
		} catch {
			message = 'Failed to reset branding.';
		}
	}

	async function uploadAsset(key: string, e: Event) {
		const target = e.target as HTMLInputElement;
		const file = target.files?.[0];
		if (!file) return;
		message = '';
		try {
			const form = new FormData();
			form.append('file', file);
			await api.upload(`/branding/assets/${key}`, form);
			message = `${key} uploaded.`;
		} catch {
			message = `Failed to upload ${key} (PNG, JPEG, GIF, WEBP, BMP, or ICO only).`;
		} finally {
			target.value = '';
		}
	}

	$effect(() => { loadConfig(); });
</script>

<h1>Branding</h1>

{#if loading}
	<p class="loading">Loading branding config...</p>
{:else if !config}
	<p>Failed to load branding configuration.</p>
{:else}
	{#if message}
		<div class="message">{message}</div>
	{/if}

	<section>
		<h2>General</h2>
		<div class="field">
			<label for="company">Company Name</label>
			<input id="company" type="text" bind:value={config.company_name} />
		</div>
		<div class="field">
			<label for="title">Document Title</label>
			<input id="title" type="text" bind:value={config.document_title} placeholder="Browser tab title" />
		</div>
		<div class="field">
			<label for="font">Font Family</label>
			<input id="font" type="text" bind:value={config.font_family} placeholder="IBM Plex Sans" />
		</div>
	</section>

	<section>
		<h2>Colors</h2>
		<div class="color-grid">
			{#each [
				['Primary', 'primary_color'],
				['Secondary', 'secondary_color'],
				['Accent', 'accent_color'],
				['Background', 'background_color'],
				['Surface', 'surface_color'],
				['Text', 'text_color'],
				['Error', 'error_color'],
				['Success', 'success_color']
			] as [label, key]}
				<div class="color-field">
					<span class="group-label">{label}</span>
					<div class="color-input">
						<input type="color" aria-label="{label} color picker" bind:value={config[key as keyof BrandingConfig]} />
						<input type="text" aria-label="{label} color value" bind:value={config[key as keyof BrandingConfig]} />
					</div>
				</div>
			{/each}
		</div>
	</section>

	<section>
		<h2>Assets</h2>
		<div class="field">
			<label for="logo-upload">Logo (PNG, JPEG, GIF, WEBP, BMP, ICO)</label>
			<input id="logo-upload" type="file" accept="image/*" onchange={(e) => uploadAsset('logo', e)} />
		</div>
		<div class="field">
			<label for="favicon-upload">Favicon</label>
			<input id="favicon-upload" type="file" accept="image/*" onchange={(e) => uploadAsset('favicon', e)} />
		</div>
	</section>

	<section>
		<h2>Custom CSS</h2>
		<textarea bind:value={config.custom_css} rows="6" placeholder="/* Custom CSS overrides */"></textarea>
	</section>

	<div class="actions">
		<button class="btn-primary" onclick={saveConfig} disabled={saving}>
			{saving ? 'Saving...' : 'Save Changes'}
		</button>
		<button class="btn-secondary" onclick={resetConfig}>Reset to Defaults</button>
	</div>
{/if}

<style>
	h1 { font-weight: 400; font-size: 1.5rem; margin-bottom: 1.5rem; }
	h2 { font-weight: 400; font-size: 1.125rem; margin-bottom: 1rem; color: var(--dd-text-secondary); }

	section { margin-bottom: 2rem; }

	.field { margin-bottom: 1rem; }
	.field label { display: block; font-size: 0.75rem; color: var(--dd-text-secondary); margin-bottom: 0.25rem; }
	.field input {
		width: 100%;
		padding: 0.5rem 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
	}

	.color-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 1rem; }
	.color-field .group-label { display: block; font-size: 0.75rem; color: var(--dd-text-secondary); margin-bottom: 0.25rem; }
	.color-input { display: flex; gap: 0.5rem; align-items: center; }
	.color-input input[type="color"] { width: 36px; height: 36px; border: none; cursor: pointer; background: none; }
	.color-input input[type="text"] {
		flex: 1;
		padding: 0.375rem 0.5rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-family: monospace;
		font-size: 0.8125rem;
	}

	textarea {
		width: 100%;
		padding: 0.75rem;
		background: var(--dd-background);
		border: 1px solid var(--dd-border);
		color: var(--dd-text);
		font-family: monospace;
		font-size: 0.8125rem;
		resize: vertical;
	}

	.actions { display: flex; gap: 1rem; }
	.btn-primary {
		padding: 0.625rem 1.25rem;
		background: var(--dd-primary);
		color: #fff;
		border: none;
		cursor: pointer;
	}
	.btn-primary:disabled { opacity: 0.5; cursor: default; }
	.btn-secondary {
		padding: 0.625rem 1.25rem;
		background: var(--dd-surface);
		color: var(--dd-text);
		border: 1px solid var(--dd-border);
		cursor: pointer;
	}

	.message {
		padding: 0.75rem;
		background: var(--dd-surface);
		border: 1px solid var(--dd-primary);
		margin-bottom: 1rem;
	}

	.loading { color: var(--dd-text-secondary); }
</style>
