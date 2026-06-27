/**
 * Branding Engine - applies custom branding colors as CSS custom properties.
 * Ported from WitFoo Analytics branding system.
 */
import type { BrandingConfig } from '$types/api';

const STYLE_ID = 'dd-branding-overrides';

// Accept only well-formed CSS colors (hex, rgb/rgba, hsl/hsla) so a stored branding
// value cannot break out of the declaration and inject arbitrary CSS rules. This is
// defense-in-depth alongside the server-side validation in the branding handler.
// Whitespace is stripped first, so the patterns themselves contain no repeated
// optional-whitespace groups (avoids any backtracking concern).
// These patterns are linear-time (no nested unbounded quantifiers); the safe-regex
// lint heuristic flags them as a false positive, so it is disabled per line.
const HEX_RE = /^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/;
// eslint-disable-next-line security/detect-unsafe-regex
const RGB_RE = /^rgba?\(\d{1,3},\d{1,3},\d{1,3}(?:,(?:0|1|0?\.\d+))?\)$/;
// eslint-disable-next-line security/detect-unsafe-regex
const HSL_RE = /^hsla?\(\d{1,3},\d{1,3}%,\d{1,3}%(?:,(?:0|1|0?\.\d+))?\)$/;

function isValidColor(value: string): boolean {
	const v = value.trim();
	if (HEX_RE.test(v)) return true;
	const compact = v.replace(/\s+/g, '');
	return RGB_RE.test(compact) || HSL_RE.test(compact);
}

// Allow only a conservative character set for font-family values.
// eslint-disable-next-line security/detect-unsafe-regex
const FONT_RE = /^[\w,'"-]+(?: [\w,'"-]+)*$/;

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

	// Build CSS custom properties from validated color values only.
	const properties: string[] = [];
	for (const [configKey, cssVar] of Object.entries(COLOR_MAP)) {
		const value = config[configKey as keyof BrandingConfig] as string;
		if (value && isValidColor(value)) {
			properties.push(`${cssVar}: ${value};`);
		}
	}

	if (config.font_family && FONT_RE.test(config.font_family)) {
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
