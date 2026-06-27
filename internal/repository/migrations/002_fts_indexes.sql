-- 002_fts_indexes.sql
-- Full-text search indexes using SQLite FTS5.
--
-- NOTE: this is an external-content FTS5 table keyed on the implicit integer rowid
-- of `documents` (documents.id is a TEXT PRIMARY KEY, so the table keeps a rowid).
-- Implicit rowids are NOT stable across VACUUM, so after any VACUUM the FTS index
-- must be rebuilt (see the 'rebuild' statement at the end of this file, which runs
-- on every boot) to keep the documents_fts->documents join correct.

CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
    name,
    description,
    tags,
    content=documents,
    content_rowid=rowid
);

-- Triggers to keep FTS in sync with documents table.

CREATE TRIGGER IF NOT EXISTS documents_fts_ai AFTER INSERT ON documents BEGIN
    INSERT INTO documents_fts(rowid, name, description, tags)
    VALUES (new.rowid, new.name, new.description, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS documents_fts_ad AFTER DELETE ON documents BEGIN
    INSERT INTO documents_fts(documents_fts, rowid, name, description, tags)
    VALUES ('delete', old.rowid, old.name, old.description, old.tags);
END;

CREATE TRIGGER IF NOT EXISTS documents_fts_au AFTER UPDATE ON documents BEGIN
    INSERT INTO documents_fts(documents_fts, rowid, name, description, tags)
    VALUES ('delete', old.rowid, old.name, old.description, old.tags);
    INSERT INTO documents_fts(rowid, name, description, tags)
    VALUES (new.rowid, new.name, new.description, new.tags);
END;

-- Repopulate the index from the content table. This runs on every boot (migrations
-- are idempotent) and is cheap for small corpora. It ensures:
--   1. rows that already existed before this migration was first applied are indexed
--      (the AFTER INSERT trigger only covers rows inserted after it exists), and
--   2. the index is consistent after a VACUUM, which can renumber implicit rowids.
INSERT INTO documents_fts(documents_fts) VALUES ('rebuild');
