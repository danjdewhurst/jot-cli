CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(created_at);
CREATE INDEX IF NOT EXISTS idx_sync_changelog_synced_note_id ON sync_changelog(synced, note_id);
