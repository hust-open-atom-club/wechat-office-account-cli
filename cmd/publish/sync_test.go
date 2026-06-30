package publish

import (
	"errors"
	"strings"
	"testing"

	internalpublish "github.com/mudongliang/weoa-cli/internal/publish"
)

type fakeSyncStore struct {
	existing  map[int64]bool
	inserted  []int64
	checkErr  error
	insertErr error
}

func (s *fakeSyncStore) HasAppMsgID(appMsgID int64) (bool, error) {
	if s.checkErr != nil {
		return false, s.checkErr
	}
	return s.existing[appMsgID], nil
}

func (s *fakeSyncStore) InsertArticle(a internalpublish.Article) (bool, error) {
	if s.insertErr != nil {
		return false, s.insertErr
	}
	s.inserted = append(s.inserted, a.AppMsgID)
	return true, nil
}

func TestSyncArticlePageProcessesWholePageBeforeStopping(t *testing.T) {
	store := &fakeSyncStore{
		existing: map[int64]bool{100: true},
	}
	articles := []internalpublish.Article{
		{AppMsgID: 101, Title: "new before known"},
		{AppMsgID: 100, Title: "known"},
		{AppMsgID: 99, Title: "new after known"},
	}

	inserted := 0
	stop, err := syncArticlePage(store, articles, func(internalpublish.Article) {
		inserted++
	})
	if err != nil {
		t.Fatalf("syncArticlePage returned error: %v", err)
	}
	if !stop {
		t.Fatal("expected page to request pagination stop after seeing an existing article")
	}
	if inserted != 2 {
		t.Fatalf("insert callback count = %d, want 2", inserted)
	}
	if len(store.inserted) != 2 || store.inserted[0] != 101 || store.inserted[1] != 99 {
		t.Fatalf("inserted IDs = %v, want [101 99]", store.inserted)
	}
}

func TestSyncArticlePageReturnsCheckError(t *testing.T) {
	store := &fakeSyncStore{
		existing: map[int64]bool{},
		checkErr: errors.New("database unavailable"),
	}

	stop, err := syncArticlePage(store, []internalpublish.Article{{AppMsgID: 101}}, nil)
	if err == nil {
		t.Fatal("expected check error")
	}
	if !stop {
		t.Fatal("expected pagination stop on check error")
	}
	if !strings.Contains(err.Error(), "check article 101") {
		t.Fatalf("error = %q, want article context", err.Error())
	}
}

func TestSyncArticlePageReturnsInsertError(t *testing.T) {
	store := &fakeSyncStore{
		existing:  map[int64]bool{},
		insertErr: errors.New("disk full"),
	}

	stop, err := syncArticlePage(store, []internalpublish.Article{{AppMsgID: 101}}, nil)
	if err == nil {
		t.Fatal("expected insert error")
	}
	if !stop {
		t.Fatal("expected pagination stop on insert error")
	}
	if !strings.Contains(err.Error(), "insert article 101") {
		t.Fatalf("error = %q, want article context", err.Error())
	}
}
