/**
 * Core API types for the Due Diligence Portal.
 */

export interface User {
	id: string;
	email: string;
	name: string;
	role: 'admin' | 'company_member' | 'investor';
	is_active: boolean;
	invited_by?: string;
	last_login_at?: string;
	created_at: string;
	updated_at: string;
}

export interface InviteToken {
	id: string;
	token: string;
	email: string;
	role: 'admin' | 'company_member' | 'investor';
	invited_by: string;
	expires_at: string;
	used_at?: string;
	created_at: string;
}

export interface Document {
	id: string;
	name: string;
	description?: string;
	category_id: string;
	uploaded_by: string;
	current_version: number;
	mime_type: string;
	file_size: number;
	is_archived: boolean;
	tags?: string;
	created_at: string;
	updated_at: string;
	category_name?: string;
	uploader_name?: string;
}

export interface DocumentVersion {
	id: string;
	document_id: string;
	version_number: number;
	file_size: number;
	mime_type: string;
	checksum_sha256: string;
	change_note?: string;
	uploaded_by: string;
	created_at: string;
}

export interface Category {
	id: string;
	name: string;
	slug: string;
	description?: string;
	parent_id?: string;
	sort_order: number;
	icon?: string;
	created_at: string;
	updated_at: string;
	children?: Category[];
	document_count?: number;
}

export interface AccessGrant {
	id: string;
	user_id: string;
	resource_type: 'category' | 'document';
	resource_id: string;
	access_level: 'view' | 'download' | 'upload' | 'manage';
	granted_by: string;
	expires_at?: string;
	created_at: string;
	user_email?: string;
	user_name?: string;
	resource_name?: string;
}

export interface AuditEntry {
	id: string;
	user_id?: string;
	user_email: string;
	action: string;
	resource_type?: string;
	resource_id?: string;
	resource_name?: string;
	details?: string;
	ip_address?: string;
	user_agent?: string;
	created_at: string;
}

export interface QAThread {
	id: string;
	subject: string;
	document_id?: string;
	category_id?: string;
	status: 'open' | 'answered' | 'closed';
	asked_by: string;
	assigned_to?: string;
	created_at: string;
	updated_at: string;
	asked_by_name?: string;
	assigned_to_name?: string;
	message_count?: number;
}

export interface QAMessage {
	id: string;
	thread_id: string;
	author_id: string;
	body: string;
	is_internal: boolean;
	created_at: string;
	author_name?: string;
	author_email?: string;
	author_role?: string;
}

export interface BrandingConfig {
	company_name: string;
	primary_color: string;
	secondary_color: string;
	accent_color: string;
	error_color: string;
	warning_color: string;
	success_color: string;
	info_color: string;
	background_color: string;
	surface_color: string;
	text_color: string;
	text_secondary_color: string;
	border_color: string;
	hover_color: string;
	active_color: string;
	header_color: string;
	sidebar_color: string;
	font_family?: string;
	custom_css?: string;
	document_title?: string;
}

export interface NDATemplate {
	id: string;
	name: string;
	content: string;
	is_active: boolean;
	version: number;
	created_by: string;
	created_at: string;
	updated_at: string;
}

export interface NDASignature {
	id: string;
	template_id: string;
	user_id: string;
	signer_name: string;
	signer_email: string;
	signer_company?: string;
	ip_address: string;
	signed_at: string;
	template_name?: string;
}
