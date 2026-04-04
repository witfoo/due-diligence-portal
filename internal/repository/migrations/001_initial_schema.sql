-- 001_initial_schema.sql
-- Due Diligence Portal: Core schema
-- All timestamps stored as RFC3339 text for SQLite compatibility.

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;

-- ============================================================
-- USERS & AUTHENTICATION
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    email           TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL CHECK (role IN ('admin', 'company_member', 'investor')) DEFAULT 'investor',
    is_active       INTEGER NOT NULL DEFAULT 1,
    invited_by      TEXT REFERENCES users(id),
    last_login_at   TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

CREATE TABLE IF NOT EXISTS invite_tokens (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    token           TEXT NOT NULL UNIQUE,
    email           TEXT NOT NULL,
    role            TEXT NOT NULL CHECK (role IN ('company_member', 'investor')),
    invited_by      TEXT NOT NULL REFERENCES users(id),
    expires_at      TEXT NOT NULL,
    used_at         TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_invite_tokens_token ON invite_tokens(token);

-- ============================================================
-- DOCUMENT CATEGORIES
-- ============================================================

CREATE TABLE IF NOT EXISTS categories (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    description     TEXT,
    parent_id       TEXT REFERENCES categories(id),
    sort_order      INTEGER NOT NULL DEFAULT 0,
    icon            TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_categories_parent ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);

-- ============================================================
-- DOCUMENTS (metadata)
-- ============================================================

CREATE TABLE IF NOT EXISTS documents (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name            TEXT NOT NULL,
    description     TEXT,
    category_id     TEXT NOT NULL REFERENCES categories(id),
    uploaded_by     TEXT NOT NULL REFERENCES users(id),
    current_version INTEGER NOT NULL DEFAULT 1,
    mime_type       TEXT NOT NULL,
    file_size       INTEGER NOT NULL,
    is_archived     INTEGER NOT NULL DEFAULT 0,
    tags            TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_documents_category ON documents(category_id);
CREATE INDEX IF NOT EXISTS idx_documents_uploaded_by ON documents(uploaded_by);

-- ============================================================
-- DOCUMENT VERSIONS (actual file content as BLOBs)
-- ============================================================

CREATE TABLE IF NOT EXISTS document_versions (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    document_id     TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version_number  INTEGER NOT NULL,
    file_data       BLOB NOT NULL,
    file_size       INTEGER NOT NULL,
    mime_type       TEXT NOT NULL,
    checksum_sha256 TEXT NOT NULL,
    change_note     TEXT,
    uploaded_by     TEXT NOT NULL REFERENCES users(id),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(document_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_doc_versions_document ON document_versions(document_id);

-- ============================================================
-- PERMISSIONS (granular document/category access)
-- ============================================================

CREATE TABLE IF NOT EXISTS access_grants (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource_type   TEXT NOT NULL CHECK (resource_type IN ('category', 'document')),
    resource_id     TEXT NOT NULL,
    access_level    TEXT NOT NULL CHECK (access_level IN ('view', 'download', 'upload', 'manage')),
    granted_by      TEXT NOT NULL REFERENCES users(id),
    expires_at      TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(user_id, resource_type, resource_id)
);

CREATE INDEX IF NOT EXISTS idx_access_grants_user ON access_grants(user_id);
CREATE INDEX IF NOT EXISTS idx_access_grants_resource ON access_grants(resource_type, resource_id);

-- ============================================================
-- AUDIT LOG (immutable, append-only)
-- ============================================================

CREATE TABLE IF NOT EXISTS audit_log (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id         TEXT REFERENCES users(id),
    user_email      TEXT NOT NULL,
    action          TEXT NOT NULL,
    resource_type   TEXT,
    resource_id     TEXT,
    resource_name   TEXT,
    details         TEXT,
    ip_address      TEXT,
    user_agent      TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_log_resource ON audit_log(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at);

-- ============================================================
-- Q&A WORKFLOW
-- ============================================================

CREATE TABLE IF NOT EXISTS qa_threads (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    subject         TEXT NOT NULL,
    document_id     TEXT REFERENCES documents(id),
    category_id     TEXT REFERENCES categories(id),
    status          TEXT NOT NULL CHECK (status IN ('open', 'answered', 'closed')) DEFAULT 'open',
    asked_by        TEXT NOT NULL REFERENCES users(id),
    assigned_to     TEXT REFERENCES users(id),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_qa_threads_status ON qa_threads(status);
CREATE INDEX IF NOT EXISTS idx_qa_threads_asked_by ON qa_threads(asked_by);
CREATE INDEX IF NOT EXISTS idx_qa_threads_document ON qa_threads(document_id);

CREATE TABLE IF NOT EXISTS qa_messages (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    thread_id       TEXT NOT NULL REFERENCES qa_threads(id) ON DELETE CASCADE,
    author_id       TEXT NOT NULL REFERENCES users(id),
    body            TEXT NOT NULL,
    is_internal     INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_qa_messages_thread ON qa_messages(thread_id);

-- ============================================================
-- NDA / AGREEMENT GATES
-- ============================================================

CREATE TABLE IF NOT EXISTS nda_templates (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name            TEXT NOT NULL,
    content         TEXT NOT NULL,
    is_active       INTEGER NOT NULL DEFAULT 1,
    version         INTEGER NOT NULL DEFAULT 1,
    created_by      TEXT NOT NULL REFERENCES users(id),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS nda_signatures (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    template_id     TEXT NOT NULL REFERENCES nda_templates(id),
    user_id         TEXT NOT NULL REFERENCES users(id),
    signer_name     TEXT NOT NULL,
    signer_email    TEXT NOT NULL,
    signer_company  TEXT,
    ip_address      TEXT NOT NULL,
    signed_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(template_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_nda_signatures_user ON nda_signatures(user_id);

-- ============================================================
-- BRANDING / WHITE-LABEL
-- ============================================================

CREATE TABLE IF NOT EXISTS branding_config (
    id                   TEXT PRIMARY KEY DEFAULT 'default',
    company_name         TEXT NOT NULL DEFAULT 'Company',
    primary_color        TEXT NOT NULL DEFAULT '#0f62fe',
    secondary_color      TEXT NOT NULL DEFAULT '#393939',
    accent_color         TEXT NOT NULL DEFAULT '#f1c21b',
    error_color          TEXT NOT NULL DEFAULT '#da1e28',
    warning_color        TEXT NOT NULL DEFAULT '#f1c21b',
    success_color        TEXT NOT NULL DEFAULT '#24a148',
    info_color           TEXT NOT NULL DEFAULT '#4589ff',
    background_color     TEXT NOT NULL DEFAULT '#161616',
    surface_color        TEXT NOT NULL DEFAULT '#262626',
    text_color           TEXT NOT NULL DEFAULT '#f4f4f4',
    text_secondary_color TEXT NOT NULL DEFAULT '#c6c6c6',
    border_color         TEXT NOT NULL DEFAULT '#393939',
    hover_color          TEXT NOT NULL DEFAULT '#353535',
    active_color         TEXT NOT NULL DEFAULT '#525252',
    header_color         TEXT NOT NULL DEFAULT '#161616',
    sidebar_color        TEXT NOT NULL DEFAULT '#1c1c1c',
    font_family          TEXT,
    custom_css           TEXT,
    document_title       TEXT,
    updated_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS branding_assets (
    key             TEXT PRIMARY KEY,
    file_data       BLOB NOT NULL,
    mime_type       TEXT NOT NULL,
    file_size       INTEGER NOT NULL,
    checksum_sha256 TEXT NOT NULL,
    uploaded_by     TEXT NOT NULL REFERENCES users(id),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- ============================================================
-- ACTIVITY ANALYTICS
-- ============================================================

CREATE TABLE IF NOT EXISTS view_events (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id         TEXT NOT NULL REFERENCES users(id),
    document_id     TEXT NOT NULL REFERENCES documents(id),
    duration_ms     INTEGER,
    page_count      INTEGER,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_view_events_user ON view_events(user_id);
CREATE INDEX IF NOT EXISTS idx_view_events_document ON view_events(document_id);
CREATE INDEX IF NOT EXISTS idx_view_events_created ON view_events(created_at);

-- ============================================================
-- WATERMARK CONFIGURATION
-- ============================================================

CREATE TABLE IF NOT EXISTS watermark_config (
    id              TEXT PRIMARY KEY DEFAULT 'default',
    enabled         INTEGER NOT NULL DEFAULT 0,
    text_template   TEXT NOT NULL DEFAULT '{{user_email}} - {{date}}',
    position        TEXT NOT NULL DEFAULT 'diagonal' CHECK (position IN ('diagonal', 'top', 'bottom', 'center')),
    opacity         REAL NOT NULL DEFAULT 0.15,
    font_size       INTEGER NOT NULL DEFAULT 12,
    color           TEXT NOT NULL DEFAULT '#888888',
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- ============================================================
-- EMAIL NOTIFICATION PREFERENCES
-- ============================================================

CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id         TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email_qa        INTEGER NOT NULL DEFAULT 1,
    email_documents INTEGER NOT NULL DEFAULT 1,
    email_nda       INTEGER NOT NULL DEFAULT 1,
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
