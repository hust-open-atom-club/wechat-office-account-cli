package publish

import (
	"encoding/json"
	"strings"
	"testing"

	internalpublish "github.com/mudongliang/weoa-cli/internal/publish"
)

func TestFilterDeletedArticles(t *testing.T) {
	articles := []internalpublish.Article{
		{AppMsgID: 1, Title: "Active"},
		{AppMsgID: 2, Title: "Deleted", IsDeleted: true},
		{AppMsgID: 1, Title: "Active duplicate"},
		{AppMsgID: 3, Title: "Active 2"},
	}

	filtered := filterDeletedArticles(articles)
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}
	if filtered[0].AppMsgID != 1 || filtered[1].AppMsgID != 3 {
		t.Fatalf("filtered IDs = [%d, %d], want [1, 3]", filtered[0].AppMsgID, filtered[1].AppMsgID)
	}
}

func TestListArticleJSONOmitsDeletedField(t *testing.T) {
	payload := []listArticleJSON{
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

func TestFormatPublishDate(t *testing.T) {
	if got := formatPublishDate(0); got != "-" {
		t.Fatalf("formatPublishDate(0) = %q, want -", got)
	}
	if got := formatPublishDate(1691672129); got != "2023-08-10" {
		t.Fatalf("formatPublishDate(1691672129) = %q, want 2023-08-10", got)
	}
}

func TestFormatRemoteListSummary(t *testing.T) {
	got := formatRemoteListSummary(386, 17, 369)
	want := "\nDone. Detected 386 published records, 17 deleted, 369 articles listed.\n"
	if got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}
}
