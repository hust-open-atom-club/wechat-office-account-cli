package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/mudongliang/weoa-cli/internal/publish"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := openTestDB(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func openTestDB(path string) (*Store, error) {
	_ = os.Remove(path) // Clean slate
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	if err := migrate(sqlDB); err != nil {
		return nil, err
	}
	return &Store{db: sqlDB}, nil
}

func TestStore_InsertAndHasArticle(t *testing.T) {
	store := testStore(t)

	a := publish.Article{
		AppMsgID:    1001,
		PublishID:   60001,
		Title:       "Test Article",
		URL:         "https://mp.weixin.qq.com/s/test123",
		PublishTime: time.Now().Unix(),
		ReadNum:     50,
		LikeNum:     3,
	}

	exists, err := store.HasAppMsgID(a.AppMsgID)
	if err != nil {
		t.Fatalf("HasAppMsgID: %v", err)
	}
	if exists {
		t.Error("article should not exist before insert")
	}

	inserted, err := store.InsertArticle(a)
	if err != nil {
		t.Fatalf("InsertArticle: %v", err)
	}
	if !inserted {
		t.Error("InsertArticle should return true for new article")
	}

	exists, err = store.HasAppMsgID(a.AppMsgID)
	if err != nil {
		t.Fatalf("HasAppMsgID after insert: %v", err)
	}
	if !exists {
		t.Error("article should exist after insert")
	}

	// Insert again — should be ignored (UNIQUE on appmsgid)
	inserted, err = store.InsertArticle(a)
	if err != nil {
		t.Fatalf("InsertArticle duplicate: %v", err)
	}
	if inserted {
		t.Error("InsertArticle should return false for duplicate article")
	}
}

func TestStore_ArticleCount(t *testing.T) {
	store := testStore(t)

	count, err := store.ArticleCount()
	if err != nil {
		t.Fatalf("ArticleCount: %v", err)
	}
	if count != 0 {
		t.Errorf("initial count = %d, want 0", count)
	}

	for i := int64(1); i <= 5; i++ {
		store.InsertArticle(publish.Article{
			AppMsgID:    i,
			Title:       "Article",
			URL:         "https://example.com",
			PublishTime: time.Now().Unix(),
		})
	}

	count, err = store.ArticleCount()
	if err != nil {
		t.Fatalf("ArticleCount after inserts: %v", err)
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
}

func TestStore_ListArticles(t *testing.T) {
	store := testStore(t)

	baseTime := time.Now().Unix()
	articles := []publish.Article{
		{AppMsgID: 1, PublishID: 10, Title: "Oldest", URL: "url1", PublishTime: baseTime - 200, ReadNum: 1},
		{AppMsgID: 2, PublishID: 20, Title: "Middle", URL: "url2", PublishTime: baseTime - 100, ReadNum: 2},
		{AppMsgID: 3, PublishID: 30, Title: "Newest", URL: "url3", PublishTime: baseTime, ReadNum: 3},
	}
	for _, a := range articles {
		store.InsertArticle(a)
	}

	result, err := store.ListArticles(2)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	// DESC order: newest first
	if result[0].Title != "Newest" {
		t.Errorf("result[0] = %q, want 'Newest'", result[0].Title)
	}
	if result[1].Title != "Middle" {
		t.Errorf("result[1] = %q, want 'Middle'", result[1].Title)
	}
}

func TestStore_InsertArticle_NullableFields(t *testing.T) {
	store := testStore(t)

	a := publish.Article{
		AppMsgID:    99999,
		Title:       "No Cover or Digest",
		URL:         "https://example.com",
		PublishTime: time.Now().Unix(),
	}

	inserted, err := store.InsertArticle(a)
	if err != nil {
		t.Fatalf("InsertArticle with empty fields: %v", err)
	}
	if !inserted {
		t.Error("should insert article with empty optional fields")
	}

	result, err := store.ListArticles(1)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if result[0].Cover != "" {
		t.Errorf("Cover = %q, want empty", result[0].Cover)
	}
	if result[0].Digest != "" {
		t.Errorf("Digest = %q, want empty", result[0].Digest)
	}
}

func TestStore_InsertArticle_FullFields(t *testing.T) {
	store := testStore(t)

	a := publish.Article{
		AppMsgID:    88888,
		PublishID:   70001,
		Title:       "Full Article",
		URL:         "https://mp.weixin.qq.com/s/full123",
		PublishTime: 1720000000,
		Cover:       "https://mmbiz.qpic.cn/cover.jpg",
		Digest:      "A complete article summary",
		ReadNum:     1234,
		LikeNum:     56,
		IsDeleted:   false,
	}

	inserted, err := store.InsertArticle(a)
	if err != nil {
		t.Fatalf("InsertArticle full: %v", err)
	}
	if !inserted {
		t.Error("should insert full article")
	}

	result, err := store.ListArticles(1)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	got := result[0]
	if got.AppMsgID != a.AppMsgID {
		t.Errorf("AppMsgID = %d, want %d", got.AppMsgID, a.AppMsgID)
	}
	if got.PublishID != a.PublishID {
		t.Errorf("PublishID = %d, want %d", got.PublishID, a.PublishID)
	}
	if got.ReadNum != a.ReadNum {
		t.Errorf("ReadNum = %d, want %d", got.ReadNum, a.ReadNum)
	}
	if got.LikeNum != a.LikeNum {
		t.Errorf("LikeNum = %d, want %d", got.LikeNum, a.LikeNum)
	}
	if got.IsDeleted != a.IsDeleted {
		t.Errorf("IsDeleted = %v, want %v", got.IsDeleted, a.IsDeleted)
	}
}

func TestStore_HasAppMsgID_NotFound(t *testing.T) {
	store := testStore(t)
	exists, err := store.HasAppMsgID(123456789)
	if err != nil {
		t.Fatalf("HasAppMsgID: %v", err)
	}
	if exists {
		t.Error("should not exist for never-inserted ID")
	}
}

func TestStore_DeletedArticle(t *testing.T) {
	store := testStore(t)

	a := publish.Article{
		AppMsgID:    555,
		Title:       "Deleted Article",
		URL:         "https://example.com/deleted",
		PublishTime: time.Now().Unix(),
		IsDeleted:   true,
	}

	store.InsertArticle(a)

	result, err := store.ListArticles(1)
	if err != nil {
		t.Fatalf("ListArticles: %v", err)
	}
	if !result[0].IsDeleted {
		t.Error("IsDeleted should be true")
	}
}
