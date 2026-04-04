-- 002_fts_indexes.sql
-- Full-text search indexes using SQLite FTS5.

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
