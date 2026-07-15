package publish

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	internalpublish "github.com/mudongliang/weoa-cli/internal/publish"
)

type fakeExportStore struct {
	listLimit    int
	searchQuery  string
	searchLimit  int
	listCalled   bool
	searchCalled bool
	closeCalled  bool
	articles     []internalpublish.Article
	listErr      error
	searchErr    error
}

var _ exportStore = (*fakeExportStore)(nil)

func (s *fakeExportStore) ListActiveArticles(limit int) ([]internalpublish.Article, error) {
	s.listCalled = true
	s.listLimit = limit
	return s.articles, s.listErr
}

func (s *fakeExportStore) SearchActiveArticles(query string, limit int) ([]internalpublish.Article, error) {
	s.searchCalled = true
	s.searchQuery = query
	s.searchLimit = limit
	return s.articles, s.searchErr
}

func (s *fakeExportStore) Close() error {
	s.closeCalled = true
	return nil
}

func TestExportArticleJSONOmitsInternalFields(t *testing.T) {
	payload := []exportArticleJSON{
		{
			Title:       "Active",
			URL:         "https://example.com",
			PublishTime: "2023-08-10",
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), "is_deleted") {
		t.Fatalf("JSON should not contain is_deleted: %s", data)
	}
	if strings.Contains(string(data), "publish_id") {
		t.Fatalf("JSON should not contain publish_id: %s", data)
	}
	if strings.Contains(string(data), "appmsgid") {
		t.Fatalf("JSON should not contain appmsgid: %s", data)
	}
	if !strings.Contains(string(data), "read_num") {
		t.Fatalf("JSON should contain read_num even when zero: %s", data)
	}
	if !strings.Contains(string(data), "like_num") {
		t.Fatalf("JSON should contain like_num even when zero: %s", data)
	}
}

func TestLoadExportArticlesDefaultsToAllActive(t *testing.T) {
	store := &fakeExportStore{articles: []internalpublish.Article{{Title: "A"}}}
	restoreExportState := stubExportStore(t, store)
	defer restoreExportState()

	exportLimit = 0
	exportSearch = ""
	articles, err := loadExportArticles()
	if err != nil {
		t.Fatalf("loadExportArticles: %v", err)
	}

	if len(articles) != 1 || articles[0].Title != "A" {
		t.Fatalf("articles = %+v", articles)
	}
	if !store.listCalled || store.searchCalled || store.listLimit != 0 || !store.closeCalled {
		t.Fatalf("store calls = %+v", store)
	}
}

func TestLoadExportArticlesSearchesWithLimit(t *testing.T) {
	store := &fakeExportStore{articles: []internalpublish.Article{{Title: "Kernel"}}}
	restoreExportState := stubExportStore(t, store)
	defer restoreExportState()

	exportLimit = 5
	exportSearch = "kernel"
	articles, err := loadExportArticles()
	if err != nil {
		t.Fatalf("loadExportArticles: %v", err)
	}

	if len(articles) != 1 || articles[0].Title != "Kernel" {
		t.Fatalf("articles = %+v", articles)
	}
	if store.listCalled || !store.searchCalled || store.searchQuery != "kernel" || store.searchLimit != 5 || !store.closeCalled {
		t.Fatalf("store calls = %+v", store)
	}
}

func TestLoadExportArticlesWrapsOpenError(t *testing.T) {
	restoreNewExportStore := newExportStore
	restoreLimit := exportLimit
	restoreSearch := exportSearch
	defer func() {
		newExportStore = restoreNewExportStore
		exportLimit = restoreLimit
		exportSearch = restoreSearch
	}()

	newExportStore = func() (exportStore, error) {
		return nil, fmt.Errorf("boom")
	}

	_, err := loadExportArticles()
	if err == nil || !strings.Contains(err.Error(), "open database: boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestEscapeMarkdownTable(t *testing.T) {
	got := escapeMarkdownTable("a|b\\c\nd")
	want := `a\|b\\c d`
	if got != want {
		t.Fatalf("escapeMarkdownTable = %q, want %q", got, want)
	}
}

func TestPrintExportJSON(t *testing.T) {
	var buf bytes.Buffer
	articles := []internalpublish.Article{{
		Title:       "Article",
		URL:         "https://example.com/a",
		PublishTime: 1691672129,
		ReadNum:     12,
		LikeNum:     3,
	}}

	if err := printExportJSON(&buf, articles); err != nil {
		t.Fatalf("printExportJSON: %v", err)
	}

	var got []exportArticleJSON
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got) != 1 || got[0].PublishTime != "2023-08-10" || got[0].LikeNum != 3 {
		t.Fatalf("JSON output = %+v", got)
	}
}

func TestPrintExportCSV(t *testing.T) {
	var buf bytes.Buffer
	articles := []internalpublish.Article{{
		Title:       "A, B",
		URL:         "https://example.com/a",
		PublishTime: 1691672129,
		Digest:      "digest",
		ReadNum:     12,
		LikeNum:     3,
	}}

	if err := printExportCSV(&buf, articles); err != nil {
		t.Fatalf("printExportCSV: %v", err)
	}

	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("CSV rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "publish_time" || rows[1][0] != "2023-08-10" || rows[1][1] != "A, B" {
		t.Fatalf("CSV output = %#v", rows)
	}
}

func TestPrintExportMarkdown(t *testing.T) {
	var buf bytes.Buffer
	articles := []internalpublish.Article{{
		Title:       "A|B",
		URL:         "https://example.com/a",
		PublishTime: 1691672129,
		Digest:      "line 1\nline 2",
		ReadNum:     12,
		LikeNum:     3,
	}}

	printExportMarkdown(&buf, articles)

	output := buf.String()
	if !strings.Contains(output, "| 2023-08-10 | A\\|B | https://example.com/a | line 1 line 2 | 12 | 3 |") {
		t.Fatalf("Markdown output = %q", output)
	}
}

func stubExportStore(t *testing.T, store *fakeExportStore) func() {
	t.Helper()

	restoreNewExportStore := newExportStore
	restoreLimit := exportLimit
	restoreSearch := exportSearch
	newExportStore = func() (exportStore, error) {
		return store, nil
	}

	return func() {
		newExportStore = restoreNewExportStore
		exportLimit = restoreLimit
		exportSearch = restoreSearch
	}
}
