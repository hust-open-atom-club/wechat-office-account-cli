package publish

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// buildAPIResponse builds a realistic appmsgpublish response with proper JSON encoding.
func buildAPIResponse(records []publishRecord, totalCount int) string {
	page := publishPage{
		TotalCount:  totalCount,
		PublishList: records,
	}
	pageBytes, _ := json.Marshal(page)

	outer := appmsgpublishResponse{
		BaseResp:    baseResp{Ret: 0, ErrMsg: "ok"},
		IsAdmin:     true,
		PublishPage: string(pageBytes),
	}
	outerBytes, _ := json.Marshal(outer)
	return string(outerBytes)
}

func makeRecord(msgID, appmsgID int64, title, contentURL string) publishRecord {
	info := publishInfo{
		Type:  9,
		MsgID: msgID,
		SentInfo: sentInfo{
			Time: 1718000000,
		},
		AppMsgInfo: []appMsgItem{
			{
				AppMsgID:   appmsgID,
				ContentURL: contentURL,
				Title:      title,
				ReadNum:    10,
				LikeNum:    1,
			},
		},
	}
	infoBytes, _ := json.Marshal(info)
	return publishRecord{
		PublishType: 101,
		PublishInfo: string(infoBytes),
	}
}

func TestService_List_Success(t *testing.T) {
	records := []publishRecord{
		makeRecord(60001, 1001, "Article A", "https://mp.weixin.qq.com/s/aaa"),
		makeRecord(60000, 1000, "Article B", "https://mp.weixin.qq.com/s/bbb"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, buildAPIResponse(records, 100))
	}))
	defer server.Close()

	c := &mockClient{baseURL: server.URL}
	svc := NewService(c)

	result, err := svc.List(0, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if result.TotalCount != 100 {
		t.Errorf("TotalCount = %d, want 100", result.TotalCount)
	}
	if len(result.Articles) != 2 {
		t.Fatalf("len(Articles) = %d, want 2", len(result.Articles))
	}
	if result.Articles[0].Title != "Article A" {
		t.Errorf("Article[0].Title = %q", result.Articles[0].Title)
	}
}

func TestService_List_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := &mockClient{baseURL: server.URL}
	svc := NewService(c)

	_, err := svc.List(0, 10)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestService_List_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `not json`)
	}))
	defer server.Close()

	c := &mockClient{baseURL: server.URL}
	svc := NewService(c)

	_, err := svc.List(0, 10)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestService_List_Pagination(t *testing.T) {
	pageCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := r.URL.Query().Get("begin")
		pageCalls++

		switch begin {
		case "0":
			records := []publishRecord{
				makeRecord(10, 1001, "Page1-1", "u1"),
				makeRecord(9, 1002, "Page1-2", "u2"),
			}
			fmt.Fprint(w, buildAPIResponse(records, 3))
		case "10":
			records := []publishRecord{
				makeRecord(8, 1003, "Page2-1", "u3"),
			}
			fmt.Fprint(w, buildAPIResponse(records, 3))
		default:
			fmt.Fprint(w, buildAPIResponse(nil, 3))
		}
	}))
	defer server.Close()

	c := &mockClient{baseURL: server.URL}
	svc := NewService(c)

	all, err := svc.ListAll(nil)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 articles, got %d", len(all))
	}
	if all[0].Title != "Page1-1" {
		t.Errorf("all[0] = %q", all[0].Title)
	}
	if all[2].Title != "Page2-1" {
		t.Errorf("all[2] = %q", all[2].Title)
	}
	if pageCalls != 3 {
		t.Errorf("expected 3 page calls, got %d", pageCalls)
	}
}

func TestService_List_EarlyStop(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		records := []publishRecord{
			makeRecord(60001, 1001, "Known", "u1"),
		}
		fmt.Fprint(w, buildAPIResponse(records, 100))
	}))
	defer server.Close()

	c := &mockClient{baseURL: server.URL}
	svc := NewService(c)

	all, err := svc.ListAll(func(articles []Article) bool {
		return true // stop after first page
	})
	if err != nil {
		t.Fatalf("ListAll with early stop: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 article, got %d", len(all))
	}
	if callCount != 1 {
		t.Errorf("expected 1 API call, got %d", callCount)
	}
}
