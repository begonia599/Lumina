package database

import (
	"context"
	"log"
)

// Migrate runs idempotent database migrations.
// This covers both fresh installs and upgrades from the pre-auth schema.
func Migrate() error {
	ctx := context.Background()

	queries := []string{
		// ---- Phase 0: auth tables ----
		`CREATE TABLE IF NOT EXISTS users (
			id            SERIAL PRIMARY KEY,
			username      VARCHAR(256) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL DEFAULT '',
			provider      VARCHAR(32)  NOT NULL DEFAULT 'local',
			external_id   VARCHAR(128),
			created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider_external
		    ON users(provider, external_id)
		    WHERE external_id IS NOT NULL`,

		`CREATE TABLE IF NOT EXISTS sessions (
			id         VARCHAR(64) PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at)`,

		// ---- Existing domain tables (create-if-missing, unchanged schema
		//      for fresh installs). Columns added in a later block. ----
		`CREATE TABLE IF NOT EXISTS books (
			id            SERIAL PRIMARY KEY,
			title         VARCHAR(255) NOT NULL,
			filename      VARCHAR(255) NOT NULL,
			file_path     TEXT NOT NULL,
			file_size     BIGINT DEFAULT 0,
			encoding      VARCHAR(32) DEFAULT 'utf-8',
			chapter_count INTEGER DEFAULT 0,
			created_at    TIMESTAMPTZ DEFAULT NOW(),
			updated_at    TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS chapters (
			id          SERIAL PRIMARY KEY,
			book_id     INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
			chapter_idx INTEGER NOT NULL,
			title       VARCHAR(500) NOT NULL,
			start_pos   INTEGER NOT NULL,
			end_pos     INTEGER NOT NULL,
			UNIQUE(book_id, chapter_idx)
		)`,
		`CREATE TABLE IF NOT EXISTS reading_progress (
			id          SERIAL PRIMARY KEY,
			book_id     INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
			chapter_idx INTEGER DEFAULT 0,
			scroll_pos  DOUBLE PRECISION DEFAULT 0,
			percentage  DOUBLE PRECISION DEFAULT 0,
			updated_at  TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS bookmarks (
			id          SERIAL PRIMARY KEY,
			book_id     INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
			chapter_idx INTEGER NOT NULL,
			scroll_pos  DOUBLE PRECISION DEFAULT 0,
			note        TEXT,
			created_at  TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS user_settings (
			id         SERIAL PRIMARY KEY,
			key        VARCHAR(64) NOT NULL,
			value      JSONB NOT NULL,
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// ---- Phase 0: add user_id scoping to domain tables (idempotent) ----
		// user_id is NULLABLE so the migration works on tables that already
		// have pre-auth rows without a default admin user. The service layer
		// enforces user_id on every INSERT, and every SELECT filters by it,
		// so any orphaned NULL rows are simply invisible going forward.
		`ALTER TABLE books            ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE`,
		`ALTER TABLE reading_progress ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE`,
		`ALTER TABLE bookmarks        ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE`,
		`ALTER TABLE user_settings    ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE`,

		// Progress now uses {chapter_idx, char_offset}. Keep scroll_pos for
		// legacy compatibility; char_offset is the authoritative field.
		`ALTER TABLE reading_progress ADD COLUMN IF NOT EXISTS char_offset INTEGER NOT NULL DEFAULT 0`,

		// Bookmarks also gain char_offset for precise positioning within a chapter.
		`ALTER TABLE bookmarks        ADD COLUMN IF NOT EXISTS char_offset INTEGER NOT NULL DEFAULT 0`,

		// ---- Phase 0: uniqueness migration for user_settings ----
		// Old schema had UNIQUE(key) globally; multi-user needs (user_id, key).
		`ALTER TABLE user_settings DROP CONSTRAINT IF EXISTS user_settings_key_key`,
		`DROP INDEX IF EXISTS user_settings_key_key`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_settings_user_key
		    ON user_settings(user_id, key)`,

		// reading_progress used to have UNIQUE(book_id); per-user ownership
		// is already captured by books.user_id cascading, but we still want
		// exactly one progress row per (book,user).
		`ALTER TABLE reading_progress DROP CONSTRAINT IF EXISTS reading_progress_book_id_key`,
		`DROP INDEX IF EXISTS reading_progress_book_id_key`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_progress_book_user
		    ON reading_progress(book_id, user_id)`,

		// ---- Phase A: editable book metadata (author / description / cover / tags) ----
		`ALTER TABLE books ADD COLUMN IF NOT EXISTS author      VARCHAR(128) NOT NULL DEFAULT ''`,
		`ALTER TABLE books ADD COLUMN IF NOT EXISTS description TEXT         NOT NULL DEFAULT ''`,
		`ALTER TABLE books ADD COLUMN IF NOT EXISTS cover_path  VARCHAR(512)`,
		`ALTER TABLE books ADD COLUMN IF NOT EXISTS tags        TEXT[]       NOT NULL DEFAULT '{}'`,
		// GIN index so "books tagged X" queries stay fast as tags grow.
		`CREATE INDEX IF NOT EXISTS idx_books_tags ON books USING GIN(tags)`,

		// ---- Indexes for user-scoped queries ----
		`CREATE INDEX IF NOT EXISTS idx_books_user        ON books(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chapters_book_id  ON chapters(book_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bookmarks_user    ON bookmarks(user_id, book_id)`,
		`CREATE INDEX IF NOT EXISTS idx_progress_user     ON reading_progress(user_id, book_id)`,
	}

	for _, q := range queries {
		if _, err := Pool.Exec(ctx, q); err != nil {
			return err
		}
	}

	log.Println("[DB] Database migrations completed")
	return nil
}
