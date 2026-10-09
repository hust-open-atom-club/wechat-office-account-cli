package publish

import (
	"encoding/json"
	"errors"
	"github.com/go-resty/resty/v2"
	"net/http"
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

type fakeSyncStateStore struct {
	fakeSyncStore
	pending     bool
	markErr     error
	completeErr error
}

var _ syncStateStore = (*fakeSyncStateStore)(nil)

func (s *fakeSyncStateStore) ActiveArticleCount() (int, error) { return len(s.existing), nil }
func (s *fakeSyncStateStore) NeedsFullSync() (bool, error)     { return s.pending, nil }
func (s *fakeSyncStateStore) MarkFullSyncNeeded() error {
	if s.markErr != nil {
		return s.markErr
	}
	s.pending = true
	return nil
}
func (s *fakeSyncStateStore) MarkFullSyncComplete() error {
	if s.completeErr != nil {
		return s.completeErr
	}
	s.pending = false
	return nil
}
func (s *fakeSyncStateStore) InsertArticle(a internalpublish.Article) (bool, error) {
	if !s.pending {
		return false, errors.New("write before retry marker")
	}
	inserted, err := s.fakeSyncStore.InsertArticle(a)
	if inserted {
		s.existing[identityOf(a)] = true
	}
	return inserted, err
}

type fakeArticleLister struct {
	pages  [][]internalpublish.Article
	endErr error
	calls  int
}

var _ articleLister = (*fakeArticleLister)(nil)

func (l *fakeArticleLister) ListAll(stop func([]internalpublish.Article) bool) ([]internalpublish.Article, error) {
	var all []internalpublish.Article
	for _, page := range l.pages {
		l.calls++
		all = append(all, page...)
		if stop(page) {
			return all, nil
		}
	}
	return all, l.endErr
}

func TestSyncArticlesFailedRunRetriesAllPages(t *testing.T) {
	for _, incremental := range []bool{false, true} {
		t.Run(map[bool]string{false: "first sync", true: "incremental sync"}[incremental], func(t *testing.T) {
			store := &fakeSyncStateStore{fakeSyncStore: fakeSyncStore{existing: map[articleIdentity]bool{}}}
			if incremental {
				store.existing[articleIdentity{appMsgID: 1, url: "old"}] = true
			}
			firstPage := []internalpublish.Article{{AppMsgID: 100, URL: "new"}}
			failed := &fakeArticleLister{pages: [][]internalpublish.Article{firstPage}, endErr: errors.New("page request failed")}
			if _, err := syncArticles(store, failed, nil); err == nil {
				t.Fatal("expected request failure")
			}
			if !store.pending {
				t.Fatal("failed sync must retain retry marker")
			}
			if !store.existing[identityOf(firstPage[0])] {
				t.Fatal("expected partially persisted first page")
			}
			retry := &fakeArticleLister{pages: [][]internalpublish.Article{firstPage, {{AppMsgID: 90, URL: "older missing"}}}}
			stats, err := syncArticles(store, retry, nil)
			if err != nil {
				t.Fatal(err)
			}
			if retry.calls != 2 || stats.inserted != 1 || !store.existing[articleIdentity{appMsgID: 90, url: "older missing"}] {
				t.Fatalf("retry did not recover older page: calls=%d stats=%+v", retry.calls, stats)
			}
			if store.pending {
				t.Fatal("successful full retry must clear marker")
			}
		})
	}
}

func TestSyncArticlesIncrementalStopsAfterKnownPage(t *testing.T) {
	store := &fakeSyncStateStore{fakeSyncStore: fakeSyncStore{existing: map[articleIdentity]bool{{appMsgID: 100, url: "known"}: true}}}
	lister := &fakeArticleLister{pages: [][]internalpublish.Article{{{AppMsgID: 100, URL: "known"}}, {{AppMsgID: 90, URL: "older"}}}}
	if _, err := syncArticles(store, lister, nil); err != nil {
		t.Fatal(err)
	}
	if lister.calls != 1 || store.pending {
		t.Fatalf("successful incremental sync: calls=%d pending=%v", lister.calls, store.pending)
	}
}

func TestSyncArticlesStorageFailureStopsFullSync(t *testing.T) {
	store := &fakeSyncStateStore{fakeSyncStore: fakeSyncStore{existing: map[articleIdentity]bool{}, insertErr: errors.New("disk full")}}
	lister := &fakeArticleLister{pages: [][]internalpublish.Article{{{AppMsgID: 100, URL: "new"}}, {{AppMsgID: 90, URL: "older"}}}}
	_, err := syncArticles(store, lister, nil)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("expected insert failure, got %v", err)
	}
	if lister.calls != 1 || !store.pending {
		t.Fatalf("failed full sync: calls=%d pending=%v", lister.calls, store.pending)
	}
}

func TestSyncArticlesMarkerErrors(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(map[bool]string{false: "before writes", true: "completion"}[complete], func(t *testing.T) {
			store := &fakeSyncStateStore{fakeSyncStore: fakeSyncStore{existing: map[articleIdentity]bool{}}}
			if complete {
				store.completeErr = errors.New("state write failed")
			} else {
				store.markErr = errors.New("state write failed")
			}
			lister := &fakeArticleLister{pages: [][]internalpublish.Article{{{AppMsgID: 100, URL: "new"}}}}
			if _, err := syncArticles(store, lister, nil); err == nil {
				t.Fatal("expected marker write error")
			}
			if complete && !store.pending {
				t.Fatal("completion failure must retain retry marker")
			}
			if !complete && lister.calls != 0 {
				t.Fatal("must not fetch articles without durable retry marker")
			}
		})
	}
}

type syncResponseClient struct{ malformed bool }

var _ internalpublish.PublishClient = (*syncResponseClient)(nil)

func (c *syncResponseClient) GetWithParams(_ string, params map[string]string) (*resty.Response, error) {
	var records []map[string]string
	switch params["begin"] {
	case "0", "10":
		id := int64(100)
		if params["begin"] == "10" {
			id = 90
		}
		info, err := json.Marshal(map[string]interface{}{
			"msgid": id, "sent_info": map[string]int{"time": 1},
			"appmsg_info": []map[string]interface{}{{"appmsgid": id, "content_url": "u" + params["begin"], "title": "article"}},
		})
		if err != nil {
			return nil, err
		}
		if c.malformed && params["begin"] == "10" {
			info = []byte("invalid json")
		}
		records = []map[string]string{{"publish_info": string(info)}}
	}
	page, err := json.Marshal(map[string]interface{}{"total_count": 2, "publish_list": records})
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]interface{}{"base_resp": map[string]int{"ret": 0}, "publish_page": string(page)})
	if err != nil {
		return nil, err
	}
	return (&resty.Response{RawResponse: &http.Response{StatusCode: 200}}).SetBody(body), nil
}

func TestSyncArticlesParseFailureRetainsMarkerAndRetryRecovers(t *testing.T) {
	store := &fakeSyncStateStore{fakeSyncStore: fakeSyncStore{existing: map[articleIdentity]bool{}}}
	client := &syncResponseClient{malformed: true}
	service := internalpublish.NewService(client)
	if _, err := syncArticles(store, service, nil); err == nil || !strings.Contains(err.Error(), "list page 10") {
		t.Fatalf("expected second-page parse failure, got %v", err)
	}
	if !store.pending || len(store.existing) != 1 {
		t.Fatalf("failed sync state: pending=%v cached=%d", store.pending, len(store.existing))
	}
	client.malformed = false
	stats, err := syncArticles(store, service, nil)
	if err != nil {
		t.Fatal(err)
	}
	if store.pending || len(store.existing) != 2 || stats.inserted != 1 {
		t.Fatalf("retry failed to recover article: pending=%v cached=%d stats=%+v", store.pending, len(store.existing), stats)
	}
}
