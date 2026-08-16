package publish

import (
	"errors"
	"strings"
	"testing"

	internalpublish "github.com/mudongliang/weoa-cli/internal/publish"
)

type fakeSyncStore struct {
	existing  map[articleIdentity]bool
	inserted  []articleIdentity
	checked   []articleIdentity
	checkErr  error
	insertErr error
}

func (s *fakeSyncStore) HasArticle(appMsgID int64, url string) (bool, error) {
	identity := articleIdentity{appMsgID: appMsgID, url: url}
	s.checked = append(s.checked, identity)
	if s.checkErr != nil {
		return false, s.checkErr
	}
	return s.existing[identity], nil
}

func (s *fakeSyncStore) InsertArticle(a internalpublish.Article) (bool, error) {
	if s.insertErr != nil {
		return false, s.insertErr
	}
	s.inserted = append(s.inserted, identityOf(a))
	return true, nil
}

func TestSyncArticlePageProcessesWholePageBeforeStopping(t *testing.T) {
	store := &fakeSyncStore{
		existing: map[articleIdentity]bool{{appMsgID: 100, url: "known-url"}: true},
	}
	articles := []internalpublish.Article{
		{AppMsgID: 101, URL: "new-101", Title: "new before known"},
		{AppMsgID: 100, URL: "known-url", Title: "known"},
		{AppMsgID: 99, URL: "new-99", Title: "new after known"},
	}

	inserted := 0
	stop, err := syncArticlePage(store, articles, map[articleIdentity]bool{}, func(internalpublish.Article) {
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
	if len(store.inserted) != 2 || store.inserted[0].appMsgID != 101 || store.inserted[1].appMsgID != 99 {
		t.Fatalf("inserted identities = %v, want appmsgids [101 99]", store.inserted)
	}
}

func TestSyncArticlePageReturnsCheckError(t *testing.T) {
	store := &fakeSyncStore{
		existing: map[articleIdentity]bool{},
		checkErr: errors.New("database unavailable"),
	}

	stop, err := syncArticlePage(store, []internalpublish.Article{{AppMsgID: 101, URL: "url-101"}}, map[articleIdentity]bool{}, nil)
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
		existing:  map[articleIdentity]bool{},
		insertErr: errors.New("disk full"),
	}

	stop, err := syncArticlePage(store, []internalpublish.Article{{AppMsgID: 101, URL: "url-101"}}, map[articleIdentity]bool{}, nil)
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
		existing: map[articleIdentity]bool{{appMsgID: 100, url: "url-100"}: true},
	}
	seenThisRun := map[articleIdentity]bool{{appMsgID: 100, url: "url-100"}: true}

	stop, err := syncArticlePage(store, []internalpublish.Article{
		{AppMsgID: 100, URL: "url-100", Title: "duplicate from current run"},
		{AppMsgID: 99, URL: "url-99", Title: "older new article"},
	}, seenThisRun, nil)
	if err != nil {
		t.Fatalf("syncArticlePage returned error: %v", err)
	}
	if stop {
		t.Fatal("did not expect pagination stop for an article inserted earlier in this run")
	}
	if len(store.inserted) != 1 || store.inserted[0].appMsgID != 99 {
		t.Fatalf("inserted identities = %v, want appmsgid [99]", store.inserted)
	}
	if !seenThisRun[articleIdentity{appMsgID: 99, url: "url-99"}] {
		t.Fatal("newly inserted article was not marked as seen this run")
	}
}

func TestSyncArticlePageTreatsSameAppMsgIDWithDifferentURLsAsDistinct(t *testing.T) {
	store := &fakeSyncStore{
		existing: map[articleIdentity]bool{{appMsgID: 100, url: "primary-url"}: true},
	}

	stop, err := syncArticlePage(store, []internalpublish.Article{
		{AppMsgID: 100, URL: "primary-url", Title: "primary"},
		{AppMsgID: 100, URL: "secondary-url", Title: "secondary"},
	}, map[articleIdentity]bool{}, nil)
	if err != nil {
		t.Fatalf("syncArticlePage returned error: %v", err)
	}
	if !stop {
		t.Fatal("existing primary article should request pagination stop")
	}
	if len(store.inserted) != 1 || store.inserted[0] != (articleIdentity{appMsgID: 100, url: "secondary-url"}) {
		t.Fatalf("inserted identities = %v, want secondary article", store.inserted)
	}
}

func TestSyncArticlePageChecksDeletedArticlesButDoesNotStoreThem(t *testing.T) {
	store := &fakeSyncStore{
		existing: map[articleIdentity]bool{{appMsgID: 100, url: "url-100"}: true},
	}
	seenThisRun := map[articleIdentity]bool{}

	stop, err := syncArticlePage(store, []internalpublish.Article{
		{AppMsgID: 100, URL: "url-100", Title: "deleted existing", IsDeleted: true},
		{AppMsgID: 99, URL: "url-99", Title: "deleted new", IsDeleted: true},
		{AppMsgID: 98, URL: "url-98", Title: "active"},
	}, seenThisRun, nil)
	if err != nil {
		t.Fatalf("syncArticlePage returned error: %v", err)
	}
	if !stop {
		t.Fatal("deleted existing article should still trigger pagination stop")
	}
	if len(store.checked) != 3 || store.checked[0].appMsgID != 100 || store.checked[1].appMsgID != 99 || store.checked[2].appMsgID != 98 {
		t.Fatalf("checked identities = %v, want appmsgids [100 99 98]", store.checked)
	}
	if len(store.inserted) != 1 || store.inserted[0].appMsgID != 98 {
		t.Fatalf("inserted identities = %v, want appmsgid [98]", store.inserted)
	}
	if !seenThisRun[articleIdentity{appMsgID: 98, url: "url-98"}] {
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

func TestCountDeletedArticles(t *testing.T) {
	articles := []internalpublish.Article{
		{AppMsgID: 1},
		{AppMsgID: 2, IsDeleted: true},
		{AppMsgID: 3, IsDeleted: true},
	}

	if got := countDeletedArticles(articles); got != 2 {
		t.Fatalf("countDeletedArticles = %d, want 2", got)
	}
}
