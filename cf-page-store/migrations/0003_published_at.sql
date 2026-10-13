ALTER TABLE pages ADD COLUMN published_at TEXT;
UPDATE pages SET published_at = created_at;
