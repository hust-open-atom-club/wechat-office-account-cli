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
	checked   []int64
	checkErr  error
	insertErr error
}

func (s *fakeSyncStore) HasAppMsgID(appMsgID int64) (bool, error) {
	s.checked = append(s.checked, appMsgID)
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
	stop, err := syncArticlePage(store, articles, map[int64]bool{}, func(internalpublish.Article) {
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

	stop, err := syncArticlePage(store, []internalpublish.Article{{AppMsgID: 101}}, map[int64]bool{}, nil)
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

	stop, err := syncArticlePage(store, []internalpublish.Article{{AppMsgID: 101}}, map[int64]bool{}, nil)
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

func TestSyncArticlePageDoesNotStopForArticleInsertedEarlierThisRun(t *testing.T) {
	store := &fakeSyncStore{
		existing: map[int64]bool{100: true},
	}
	seenThisRun := map[int64]bool{100: true}

	stop, err := syncArticlePage(store, []internalpublish.Article{
		{AppMsgID: 100, Title: "duplicate from current run"},
		{AppMsgID: 99, Title: "older new article"},
	}, seenThisRun, nil)
	if err != nil {
		t.Fatalf("syncArticlePage returned error: %v", err)
	}
	if stop {
		t.Fatal("did not expect pagination stop for an article inserted earlier in this run")
	}
	if len(store.inserted) != 1 || store.inserted[0] != 99 {
		t.Fatalf("inserted IDs = %v, want [99]", store.inserted)
	}
	if !seenThisRun[99] {
		t.Fatal("newly inserted article was not marked as seen this run")
	}
}

func TestSyncArticlePageSkipsDeletedArticles(t *testing.T) {
	store := &fakeSyncStore{
		existing: map[int64]bool{100: true},
	}
	seenThisRun := map[int64]bool{}

	stop, err := syncArticlePage(store, []internalpublish.Article{
		{AppMsgID: 100, Title: "deleted existing", IsDeleted: true},
		{AppMsgID: 99, Title: "deleted new", IsDeleted: true},
		{AppMsgID: 98, Title: "active"},
	}, seenThisRun, nil)
	if err != nil {
		t.Fatalf("syncArticlePage returned error: %v", err)
	}
	if stop {
		t.Fatal("deleted articles should not trigger pagination stop")
	}
	if len(store.checked) != 1 || store.checked[0] != 98 {
		t.Fatalf("checked IDs = %v, want [98]", store.checked)
	}
	if len(store.inserted) != 1 || store.inserted[0] != 98 {
		t.Fatalf("inserted IDs = %v, want [98]", store.inserted)
	}
	if !seenThisRun[98] {
		t.Fatal("active inserted article was not marked as seen this run")
	}
}

func TestShouldStopSync(t *testing.T) {
	if shouldStopSync(true, true) {
		t.Fatal("first full sync should not stop early when a page reports an existing article")
	}
	if !shouldStopSync(false, true) {
		t.Fatal("incremental sync should stop when a page reports an existing article")
	}
	if shouldStopSync(false, false) {
		t.Fatal("incremental sync should continue when no existing article was seen")
	}
}
