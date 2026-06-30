package publish

import (
	"encoding/json"
	"testing"
)

func TestParseResponse_Success(t *testing.T) {
	// Real API response format: publish_page is a JSON-encoded string,
	// and each publish_info is also a JSON-encoded string (double encoding).
	jsonData := `{
		"base_resp": {"err_msg": "ok", "ret": 0},
		"is_admin": true,
		"publish_page": "{\"total_count\":377,\"publish_count\":75,\"publish_list\":[{\"publish_type\":101,\"publish_info\":\"{\\\"type\\\":9,\\\"msgid\\\":1000000379,\\\"sent_info\\\":{\\\"time\\\":1782365746},\\\"appmsg_info\\\":[{\\\"appmsgid\\\":2247489943,\\\"content_url\\\":\\\"https:\\\\\\/\\\\\\/mp.weixin.qq.com\\\\\\/s\\\\\\/abc123\\\",\\\"title\\\":\\\"笑傲内核会议定档\\\",\\\"is_deleted\\\":false,\\\"copyright_status\\\":201,\\\"read_num\\\":67,\\\"like_num\\\":5,\\\"cover\\\":\\\"https:\\\\\\/\\\\\\/mmbiz.qpic.cn\\\\\\/cover\\\",\\\"digest\\\":\\\"会议摘要\\\"}]}\"}]}"
	}`

	articles, total, err := parseResponse([]byte(jsonData))
	if err != nil {
		t.Fatalf("parseResponse failed: %v", err)
	}

	if total != 377 {
		t.Errorf("total = %d, want 377", total)
	}
	if len(articles) != 1 {
		t.Fatalf("len(articles) = %d, want 1", len(articles))
	}

	a := articles[0]
	if a.AppMsgID != 2247489943 {
		t.Errorf("AppMsgID = %d, want 2247489943", a.AppMsgID)
	}
	if a.PublishID != 1000000379 {
		t.Errorf("PublishID = %d, want 1000000379", a.PublishID)
	}
	if a.Title != "笑傲内核会议定档" {
		t.Errorf("Title = %q", a.Title)
	}
	if a.URL != "https://mp.weixin.qq.com/s/abc123" {
		t.Errorf("URL = %q", a.URL)
	}
	if a.PublishTime != 1782365746 {
		t.Errorf("PublishTime = %d", a.PublishTime)
	}
	if a.ReadNum != 67 {
		t.Errorf("ReadNum = %d, want 67", a.ReadNum)
	}
	if a.LikeNum != 5 {
		t.Errorf("LikeNum = %d, want 5", a.LikeNum)
	}
	if a.Cover != "https://mmbiz.qpic.cn/cover" {
		t.Errorf("Cover = %q", a.Cover)
	}
	if a.Digest != "会议摘要" {
		t.Errorf("Digest = %q", a.Digest)
	}
	if a.IsDeleted {
		t.Error("IsDeleted should be false")
	}
}

func TestParseResponse_MultiArticle(t *testing.T) {
	// One publish record with multiple articles
	jsonData := `{
		"base_resp": {"err_msg": "ok", "ret": 0},
		"is_admin": true,
		"publish_page": "{\"total_count\":100,\"publish_list\":[{\"publish_type\":101,\"publish_info\":\"{\\\"type\\\":9,\\\"msgid\\\":50001,\\\"sent_info\\\":{\\\"time\\\":1000},\\\"appmsg_info\\\":[{\\\"appmsgid\\\":101,\\\"content_url\\\":\\\"https:\\\\\\/\\\\\\/mp.weixin.qq.com\\\\\\/s\\\\\\/a\\\",\\\"title\\\":\\\"Main Article\\\",\\\"is_deleted\\\":false,\\\"read_num\\\":10,\\\"like_num\\\":1},{\\\"appmsgid\\\":102,\\\"content_url\\\":\\\"https:\\\\\\/\\\\\\/mp.weixin.qq.com\\\\\\/s\\\\\\/b\\\",\\\"title\\\":\\\"Sub Article\\\",\\\"is_deleted\\\":false,\\\"read_num\\\":5,\\\"like_num\\\":0}]}\"}]}"
	}`

	articles, total, err := parseResponse([]byte(jsonData))
	if err != nil {
		t.Fatalf("parseResponse failed: %v", err)
	}
	if total != 100 {
		t.Errorf("total = %d", total)
	}
	if len(articles) != 2 {
		t.Fatalf("len(articles) = %d, want 2", len(articles))
	}
	if articles[0].Title != "Main Article" {
		t.Errorf("articles[0].Title = %q", articles[0].Title)
	}
	if articles[1].Title != "Sub Article" {
		t.Errorf("articles[1].Title = %q", articles[1].Title)
	}
	// Both should share the same publish_id
	if articles[0].PublishID != 50001 || articles[1].PublishID != 50001 {
		t.Error("both articles should have same PublishID")
	}
}

func TestParseResponse_DeletedArticle(t *testing.T) {
	jsonData := `{
		"base_resp": {"err_msg": "ok", "ret": 0},
		"is_admin": true,
		"publish_page": "{\"total_count\":1,\"publish_list\":[{\"publish_type\":101,\"publish_info\":\"{\\\"type\\\":9,\\\"msgid\\\":1,\\\"sent_info\\\":{\\\"time\\\":1},\\\"appmsg_info\\\":[{\\\"appmsgid\\\":999,\\\"content_url\\\":\\\"\\\",\\\"title\\\":\\\"Deleted Post\\\",\\\"is_deleted\\\":true,\\\"read_num\\\":0,\\\"like_num\\\":0}]}\"}]}"
	}`

	articles, _, err := parseResponse([]byte(jsonData))
	if err != nil {
		t.Fatalf("parseResponse failed: %v", err)
	}
	if !articles[0].IsDeleted {
		t.Error("IsDeleted should be true")
	}
}

func TestParseResponse_EmptyList(t *testing.T) {
	jsonData := `{
		"base_resp": {"err_msg": "ok", "ret": 0},
		"is_admin": true,
		"publish_page": "{\"total_count\":0,\"publish_list\":[]}"
	}`

	articles, total, err := parseResponse([]byte(jsonData))
	if err != nil {
		t.Fatalf("parseResponse failed: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(articles) != 0 {
		t.Errorf("expected empty articles, got %d", len(articles))
	}
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	_, _, err := parseResponse([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseResponse_InvalidPublishPage(t *testing.T) {
	// publish_page is not valid JSON
	jsonData := `{
		"base_resp": {"err_msg": "ok", "ret": 0},
		"is_admin": true,
		"publish_page": "not valid json"
	}`
	_, _, err := parseResponse([]byte(jsonData))
	if err == nil {
		t.Error("expected error for invalid publish_page JSON")
	}
}

func TestParseResponse_BaseRespError(t *testing.T) {
	jsonData := `{
		"base_resp": {"err_msg": "invalid session", "ret": 200003},
		"is_admin": false,
		"publish_page": ""
	}`
	_, _, err := parseResponse([]byte(jsonData))
	if err == nil {
		t.Fatal("expected error for non-zero base_resp ret")
	}
}

func TestParseResponse_MalformedPublishInfo(t *testing.T) {
	// publish_info contains invalid JSON — should be skipped gracefully
	jsonData := `{
		"base_resp": {"err_msg": "ok", "ret": 0},
		"is_admin": true,
		"publish_page": "{\"total_count\":2,\"publish_list\":[{\"publish_type\":101,\"publish_info\":\"{\\\"type\\\":9,\\\"msgid\\\":1,\\\"sent_info\\\":{\\\"time\\\":1},\\\"appmsg_info\\\":[{\\\"appmsgid\\\":1,\\\"content_url\\\":\\\"u1\\\",\\\"title\\\":\\\"Good\\\",\\\"is_deleted\\\":false,\\\"read_num\\\":0,\\\"like_num\\\":0}]}\"},{\"publish_type\":101,\"publish_info\":\"invalid json here\"}]}"
	}`

	articles, total, err := parseResponse([]byte(jsonData))
	if err != nil {
		t.Fatalf("parseResponse should not fail on malformed records: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d", total)
	}
	if len(articles) != 1 {
		t.Errorf("expected 1 article (1 skipped), got %d", len(articles))
	}
}

func TestArticle_JSONRoundTrip(t *testing.T) {
	a := Article{
		PublishID:   1000000379,
		AppMsgID:    2247489943,
		Title:       "测试文章",
		URL:         "https://mp.weixin.qq.com/s/test123",
		PublishTime: 1782365746,
		Cover:       "https://example.com/cover.jpg",
		Digest:      "这是一篇测试文章",
		ReadNum:     100,
		LikeNum:     10,
		IsDeleted:   false,
	}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var b Article
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if b.AppMsgID != a.AppMsgID {
		t.Errorf("AppMsgID: got %d, want %d", b.AppMsgID, a.AppMsgID)
	}
	if b.PublishID != a.PublishID {
		t.Errorf("PublishID: got %d, want %d", b.PublishID, a.PublishID)
	}
	if b.ReadNum != a.ReadNum {
		t.Errorf("ReadNum: got %d, want %d", b.ReadNum, a.ReadNum)
	}
	if b.LikeNum != a.LikeNum {
		t.Errorf("LikeNum: got %d, want %d", b.LikeNum, a.LikeNum)
	}
}
