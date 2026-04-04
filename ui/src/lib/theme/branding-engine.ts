/**
 * Branding Engine - applies custom branding colors as CSS custom properties.
 * Ported from WitFoo Analytics branding system.
 */
import type { BrandingConfig } from '$types/api';

const STYLE_ID = 'dd-branding-overrides';

const COLOR_MAP: Record<string, string> = {
	primary_color: '--dd-primary',
	secondary_color: '--dd-secondary',
	accent_color: '--dd-accent',
	error_color: '--dd-error',
	warning_color: '--dd-warning',
	success_color: '--dd-success',
	info_color: '--dd-info',
	background_color: '--dd-background',
	surface_color: '--dd-surface',
	text_color: '--dd-text',
	text_secondary_color: '--dd-text-secondary',
	border_color: '--dd-border',
	hover_color: '--dd-hover',
	active_color: '--dd-active',
	header_color: '--dd-header',
	sidebar_color: '--dd-sidebar'
};

/**
 * Apply branding configuration as CSS custom properties.
 */
export function applyBrandingCSS(config: BrandingConfig): void {
	if (typeof document === 'undefined') return;

	// Remove existing branding styles.
	const existing = document.getElementById(STYLE_ID);
	if (existing) {
		existing.remove();
	}

	// Build CSS custom properties.
	const properties: string[] = [];
	for (const [configKey, cssVar] of Object.entries(COLOR_MAP)) {
		const value = config[configKey as keyof BrandingConfig] as string;
		if (value) {
			properties.push(`${cssVar}: ${value};`);
		}
	}

	if (config.font_family) {
		properties.push(`--dd-font-family: ${config.font_family};`);
	}

	let css = `:root {\n${properties.map((p) => `  ${p}`).join('\n')}\n}`;

	// Append sanitized custom CSS if provided.
	if (config.custom_css) {
		css += `\n/* Custom CSS */\n${config.custom_css}`;
	}

	// Inject style element.
	const style = document.createElement('style');
	style.id = STYLE_ID;
	style.textContent = css;
	document.head.appendChild(style);
}

/**
 * Remove all branding overrides, reverting to defaults.
 */
export function removeBrandingCSS(): void {
	if (typeof document === 'undefined') return;
	const existing = document.getElementById(STYLE_ID);
	if (existing) {
		existing.remove();
	}
}

/**
 * Update the document title from branding config.
 */
export function applyDocumentTitle(config: BrandingConfig): void {
	if (typeof document === 'undefined') return;
	if (config.document_title) {
		document.title = config.document_title;
	} else if (config.company_name) {
		document.title = `${config.company_name} - Due Diligence Portal`;
	}
}
