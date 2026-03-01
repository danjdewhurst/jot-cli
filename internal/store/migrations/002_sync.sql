-- Track sync metadata and changes for file-based sync.

CREATE TABLE IF NOT EXISTS sync_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sync_changelog (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id    TEXT NOT NULL,
    action     TEXT NOT NULL CHECK (action IN ('upsert', 'delete')),
    changed_at TEXT NOT NULL,
    synced     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_sync_changelog_synced
    ON sync_changelog(synced, id);

-- Trigger: after INSERT on notes, log an upsert
CREATE TRIGGER IF NOT EXISTS trg_sync_note_insert
AFTER INSERT ON notes
BEGIN
    INSERT INTO sync_changelog (note_id, action, changed_at)
    VALUES (NEW.id, 'upsert', NEW.updated_at);
END;

-- Trigger: after UPDATE on notes, log an upsert
CREATE TRIGGER IF NOT EXISTS trg_sync_note_update
AFTER UPDATE ON notes
BEGIN
    INSERT INTO sync_changelog (note_id, action, changed_at)
    VALUES (NEW.id, 'upsert', NEW.updated_at);
END;

-- Trigger: after DELETE on notes, log a delete
CREATE TRIGGER IF NOT EXISTS trg_sync_note_delete
AFTER DELETE ON notes
BEGIN
    INSERT INTO sync_changelog (note_id, action, changed_at)
    VALUES (OLD.id, 'delete', OLD.updated_at);
END;

-- Trigger: after INSERT on tags, log an upsert for the parent note
CREATE TRIGGER IF NOT EXISTS trg_sync_tag_insert
AFTER INSERT ON tags
BEGIN
    INSERT INTO sync_changelog (note_id, action, changed_at)
    VALUES (NEW.note_id, 'upsert',
        (SELECT updated_at FROM notes WHERE id = NEW.note_id));
END;

-- Trigger: after DELETE on tags, log an upsert for the parent note.
-- The WHEN clause skips this when the note itself is being deleted (cascade),
-- since trg_sync_note_delete already handles that case.
CREATE TRIGGER IF NOT EXISTS trg_sync_tag_delete
AFTER DELETE ON tags
WHEN (SELECT 1 FROM notes WHERE id = OLD.note_id) IS NOT NULL
BEGIN
    INSERT INTO sync_changelog (note_id, action, changed_at)
    VALUES (OLD.note_id, 'upsert',
        (SELECT updated_at FROM notes WHERE id = OLD.note_id));
END;
