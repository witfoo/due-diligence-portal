-- ============================================================
-- NDA EXEMPTIONS
-- ============================================================
-- Per-user waiver of the NDA click-through requirement. A row here means the
-- user is not gated by /nda/status. Optionally stores an externally executed
-- NDA document (e.g. a countersigned PDF) as evidence for the waiver.

CREATE TABLE IF NOT EXISTS nda_exemptions (
    user_id         TEXT PRIMARY KEY REFERENCES users(id),
    reason          TEXT NOT NULL DEFAULT '',
    file_name       TEXT,
    mime_type       TEXT,
    file_size       INTEGER,
    file_data       BLOB,
    granted_by      TEXT NOT NULL REFERENCES users(id),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
