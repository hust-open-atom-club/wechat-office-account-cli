package publish

import (
	"testing"

	internalpublish "github.com/mudongliang/weoa-cli/internal/publish"
)

func TestFilterDeletedArticles(t *testing.T) {
	articles := []internalpublish.Article{
		{AppMsgID: 1, URL: "primary-url", Title: "Active"},
		{AppMsgID: 2, URL: "deleted-url", Title: "Deleted", IsDeleted: true},
		{AppMsgID: 1, URL: "primary-url", Title: "Exact duplicate"},
		{AppMsgID: 1, URL: "secondary-url", Title: "Same ID, different URL"},
		{AppMsgID: 3, URL: "third-url", Title: "Active 2"},
	}

	filtered := filterDeletedArticles(articles)
	if len(filtered) != 3 {
		t.Fatalf("len(filtered) = %d, want 3", len(filtered))
	}
	if filtered[0].URL != "primary-url" || filtered[1].URL != "secondary-url" || filtered[2].URL != "third-url" {
		t.Fatalf("filtered URLs = [%q, %q, %q]", filtered[0].URL, filtered[1].URL, filtered[2].URL)
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
