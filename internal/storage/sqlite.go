package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/mudongliang/weoa-cli/internal/config"
	"github.com/mudongliang/weoa-cli/internal/publish"
)

// Store wraps the SQLite database for article storage.
type Store struct {
	db *sql.DB
}

// New opens (or creates) the SQLite database and ensures the schema exists.
func New() (*Store, error) {
	if err := config.EnsureDir(); err != nil {
		return nil, fmt.Errorf("ensure config dir: %w", err)
	}

	db, err := sql.Open("sqlite", config.DBFile())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// InsertArticle inserts an article if it doesn't already exist (by appmsgid and URL).
// Returns true if the article was newly inserted.
func (s *Store) InsertArticle(a publish.Article) (bool, error) {
	result, err := s.db.Exec(
		`INSERT OR IGNORE INTO articles (appmsgid, publish_id, title, url, publish_time, cover, digest, read_num, like_num, is_deleted, synced_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.AppMsgID, a.PublishID, a.Title, a.URL, a.PublishTime, a.Cover, a.Digest, a.ReadNum, a.LikeNum, a.IsDeleted, time.Now().Unix(),
	)
	if err != nil {
		return false, fmt.Errorf("insert article: %w", err)
	}

	n, _ := result.RowsAffected()
	return n > 0, nil
}

// HasArticle returns true if the appmsgid and URL pair is already in the database.
func (s *Store) HasArticle(appMsgID int64, url string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM articles WHERE appmsgid = ? AND url = ?)",
		appMsgID, url,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check article identity: %w", err)
	}
	return exists, nil
}

// NeedsFullSync reports whether a migration or interrupted synchronization
// requires a full traversal to recover missing articles.
func (s *Store) NeedsFullSync() (bool, error) {
	var value int
	err := s.db.QueryRow("SELECT value FROM sync_state WHERE key = 'needs_full_sync'").Scan(&value)
	if err != nil {
		return false, fmt.Errorf("read full sync state: %w", err)
	}
	return value != 0, nil
}

// MarkFullSyncNeeded persists retry state before a sync can partially write articles.
func (s *Store) MarkFullSyncNeeded() error {
	_, err := s.db.Exec("UPDATE sync_state SET value = 1 WHERE key = 'needs_full_sync'")
	if err != nil {
		return fmt.Errorf("mark full sync needed: %w", err)
	}
	return nil
}

// MarkFullSyncComplete clears the retry marker after successful synchronization.
func (s *Store) MarkFullSyncComplete() error {
	_, err := s.db.Exec("UPDATE sync_state SET value = 0 WHERE key = 'needs_full_sync'")
	if err != nil {
		return fmt.Errorf("mark full sync complete: %w", err)
	}
	return nil
}

// ArticleCount returns the total number of articles in the database.
func (s *Store) ArticleCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count articles: %w", err)
	}
	return count, nil
}

// ActiveArticleCount returns the number of non-deleted articles in the database.
func (s *Store) ActiveArticleCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM articles WHERE is_deleted = 0").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active articles: %w", err)
	}
	return count, nil
}

// ListArticles returns articles ordered by appmsgid descending.
// If limit <= 0, returns all articles without limit.
func (s *Store) ListArticles(limit int) ([]publish.Article, error) {
	return s.listArticles(limit, false)
}

// ListActiveArticles returns non-deleted articles ordered by appmsgid descending.
// If limit <= 0, returns all active articles without limit.
func (s *Store) ListActiveArticles(limit int) ([]publish.Article, error) {
	return s.listArticles(limit, true)
}

// SearchActiveArticles returns non-deleted articles matching query in title, digest, or URL.
// If limit <= 0, returns all matching active articles without limit.
func (s *Store) SearchActiveArticles(query string, limit int) ([]publish.Article, error) {
	pattern, ok := searchPattern(query)
	if !ok {
		return s.ListActiveArticles(limit)
	}

	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = s.db.Query(
			`SELECT appmsgid, publish_id, title, url, publish_time, cover, digest, read_num, like_num, is_deleted
			 FROM articles
			 WHERE is_deleted = 0 AND (title LIKE ? OR digest LIKE ? OR url LIKE ?)
			 ORDER BY appmsgid DESC LIMIT ?`,
			pattern, pattern, pattern, limit,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT appmsgid, publish_id, title, url, publish_time, cover, digest, read_num, like_num, is_deleted
			 FROM articles
			 WHERE is_deleted = 0 AND (title LIKE ? OR digest LIKE ? OR url LIKE ?)
			 ORDER BY appmsgid DESC`,
			pattern, pattern, pattern,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("search articles: %w", err)
	}
	return scanArticles(rows)
}

// SearchActiveArticleCount returns the number of non-deleted articles matching query.
func (s *Store) SearchActiveArticleCount(query string) (int, error) {
	pattern, ok := searchPattern(query)
	if !ok {
		return s.ActiveArticleCount()
	}

	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM articles
		 WHERE is_deleted = 0 AND (title LIKE ? OR digest LIKE ? OR url LIKE ?)`,
		pattern, pattern, pattern,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count search articles: %w", err)
	}
	return count, nil
}

func searchPattern(query string) (string, bool) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", false
	}
	return "%" + query + "%", true
}

func (s *Store) listArticles(limit int, activeOnly bool) ([]publish.Article, error) {
	var rows *sql.Rows
	var err error

	switch {
	case activeOnly && limit > 0:
		rows, err = s.db.Query(
			`SELECT appmsgid, publish_id, title, url, publish_time, cover, digest, read_num, like_num, is_deleted
			 FROM articles WHERE is_deleted = 0 ORDER BY appmsgid DESC LIMIT ?`, limit,
		)
	case activeOnly:
		rows, err = s.db.Query(
			`SELECT appmsgid, publish_id, title, url, publish_time, cover, digest, read_num, like_num, is_deleted
			 FROM articles WHERE is_deleted = 0 ORDER BY appmsgid DESC`,
		)
	case limit > 0:
		rows, err = s.db.Query(
			`SELECT appmsgid, publish_id, title, url, publish_time, cover, digest, read_num, like_num, is_deleted
			 FROM articles ORDER BY appmsgid DESC LIMIT ?`, limit,
		)
	default:
		rows, err = s.db.Query(
			`SELECT appmsgid, publish_id, title, url, publish_time, cover, digest, read_num, like_num, is_deleted
			 FROM articles ORDER BY appmsgid DESC`,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("query articles: %w", err)
	}
	return scanArticles(rows)
}

func scanArticles(rows *sql.Rows) ([]publish.Article, error) {
	defer rows.Close()
	var articles []publish.Article
	for rows.Next() {
		var a publish.Article
		var cover, digest sql.NullString
		var readNum, likeNum sql.NullInt64
		if err := rows.Scan(&a.AppMsgID, &a.PublishID, &a.Title, &a.URL, &a.PublishTime,
			&cover, &digest, &readNum, &likeNum, &a.IsDeleted); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		if cover.Valid {
			a.Cover = cover.String
		}
		if digest.Valid {
			a.Digest = digest.String
		}
		if readNum.Valid {
			a.ReadNum = int(readNum.Int64)
		}
		if likeNum.Valid {
			a.LikeNum = int(likeNum.Int64)
		}
		articles = append(articles, a)
	}

	return articles, rows.Err()
}

func migrate(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin schema migration: %w", err)
	}
	defer tx.Rollback()

	var schemaVersion int
	if err := tx.QueryRow("PRAGMA user_version").Scan(&schemaVersion); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	var articlesExist bool
	if err := tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'articles')",
	).Scan(&articlesExist); err != nil {
		return fmt.Errorf("check articles table: %w", err)
	}

	if articlesExist && schemaVersion < 1 {
		if _, err := tx.Exec(`
			CREATE TABLE articles_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				appmsgid INTEGER NOT NULL,
				publish_id INTEGER NOT NULL,
				title TEXT NOT NULL,
				url TEXT NOT NULL,
				publish_time INTEGER NOT NULL,
				cover TEXT,
				digest TEXT,
				read_num INTEGER DEFAULT 0,
				like_num INTEGER DEFAULT 0,
				is_deleted INTEGER DEFAULT 0,
				synced_at INTEGER NOT NULL,
				UNIQUE(appmsgid, url)
			);
			INSERT OR IGNORE INTO articles_new
				(id, appmsgid, publish_id, title, url, publish_time, cover, digest, read_num, like_num, is_deleted, synced_at)
			SELECT id, appmsgid, publish_id, title, url, publish_time, cover, digest, read_num, like_num, is_deleted, synced_at
			FROM articles;
			DROP TABLE articles;
			ALTER TABLE articles_new RENAME TO articles;
		`); err != nil {
			return fmt.Errorf("migrate article identity: %w", err)
		}
	}

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			appmsgid INTEGER NOT NULL,
			publish_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			publish_time INTEGER NOT NULL,
			cover TEXT,
			digest TEXT,
			read_num INTEGER DEFAULT 0,
			like_num INTEGER DEFAULT 0,
			is_deleted INTEGER DEFAULT 0,
			synced_at INTEGER NOT NULL,
			UNIQUE(appmsgid, url)
		);
		CREATE INDEX IF NOT EXISTS idx_articles_appmsgid ON articles(appmsgid DESC);
		CREATE TABLE IF NOT EXISTS sync_state (
			key TEXT PRIMARY KEY,
			value INTEGER NOT NULL
		);
		INSERT OR IGNORE INTO sync_state (key, value) VALUES ('needs_full_sync', 0);
	`); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	if articlesExist && schemaVersion < 1 {
		if _, err := tx.Exec("UPDATE sync_state SET value = 1 WHERE key = 'needs_full_sync'"); err != nil {
			return fmt.Errorf("schedule full sync: %w", err)
		}
	}
	if _, err := tx.Exec("PRAGMA user_version = 1"); err != nil {
		return fmt.Errorf("write schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema migration: %w", err)
	}
	return nil
}
