package publish

import (
	"encoding/json"
	"fmt"
)

// Article represents a single published article.
type Article struct {
	PublishID   int64  `json:"publish_id"`
	AppMsgID    int64  `json:"appmsgid"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	PublishTime int64  `json:"publish_time"`
	Cover       string `json:"cover,omitempty"`
	Digest      string `json:"digest,omitempty"`
	ReadNum     int    `json:"read_num,omitempty"`
	LikeNum     int    `json:"like_num,omitempty"`
	IsDeleted   bool   `json:"is_deleted"`
}

// appmsgpublishResponse is the actual JSON structure returned by
// GET /cgi-bin/appmsgpublish?sub=list&f=json
//
// Note: publish_page is a JSON-encoded STRING, not a nested object.
// The inner publish_info is also a JSON-encoded STRING.
type appmsgpublishResponse struct {
	BaseResp    baseResp `json:"base_resp"`
	IsAdmin     bool     `json:"is_admin"`
	PublishPage string   `json:"publish_page"`
}

type baseResp struct {
	Ret    int    `json:"ret"`
	ErrMsg string `json:"err_msg"`
}

// publishPage is the decoded inner structure of publish_page.
type publishPage struct {
	TotalCount    int             `json:"total_count"`
	PublishCount  int             `json:"publish_count"`
	MasssendCount int             `json:"masssend_count"`
	PublishList   []publishRecord `json:"publish_list"`
}

type publishRecord struct {
	PublishType int    `json:"publish_type"`
	PublishInfo string `json:"publish_info"` // JSON-encoded string
}

// publishInfo is the decoded inner structure of publish_info.
type publishInfo struct {
	Type       int          `json:"type"`
	MsgID      int64        `json:"msgid"`
	SentInfo   sentInfo     `json:"sent_info"`
	SentStatus sentStatus   `json:"sent_status"`
	AppMsgInfo []appMsgItem `json:"appmsg_info"`
}

type sentInfo struct {
	Time        int64 `json:"time"`
	IsSendAll   bool  `json:"is_send_all"`
	IsPublished int   `json:"is_published"`
}

type sentStatus struct {
	Total    int `json:"total"`
	Succ     int `json:"succ"`
	Fail     int `json:"fail"`
	Progress int `json:"progress"`
}

type appMsgItem struct {
	AppMsgID        int64  `json:"appmsgid"`
	ContentURL      string `json:"content_url"`
	Title           string `json:"title"`
	IsDeleted       bool   `json:"is_deleted"`
	CopyrightStatus int    `json:"copyright_status"`
	CopyrightType   int    `json:"copyright_type"`
	ReadNum         int    `json:"read_num"`
	LikeNum         int    `json:"like_num"`
	Cover           string `json:"cover"`
	Digest          string `json:"digest"`
}

// parseResponse decodes the actual API response into articles.
func parseResponse(body []byte) ([]Article, int, error) {
	page, articles, err := parseArticlePage(body)
	if err != nil {
		return nil, 0, err
	}
	return articles, page.TotalCount, nil
}

// parseArticlePage retains raw publish records so a page without article items
// is not mistaken for the end of pagination.
func parseArticlePage(body []byte) (*publishPage, []Article, error) {
	var outer appmsgpublishResponse
	if err := json.Unmarshal(body, &outer); err != nil {
		return nil, nil, err
	}
	if outer.BaseResp.Ret != 0 {
		return nil, nil, fmt.Errorf("wechat api error %d: %s", outer.BaseResp.Ret, outer.BaseResp.ErrMsg)
	}

	// Decode the first layer of JSON-string: publish_page
	var page *publishPage
	if err := json.Unmarshal([]byte(outer.PublishPage), &page); err != nil {
		return nil, nil, err
	}

	if page == nil {
		return nil, nil, fmt.Errorf("publish_page is null")
	}

	articles := make([]Article, 0)
	for index, record := range page.PublishList {
		// Decode the second layer of JSON-string: publish_info
		var info *publishInfo
		if err := json.Unmarshal([]byte(record.PublishInfo), &info); err != nil {
			return nil, nil, fmt.Errorf("decode publish_info at record %d: %w", index, err)
		}

		if info == nil {
			return nil, nil, fmt.Errorf("publish_info at record %d is null", index)
		}

		for _, item := range info.AppMsgInfo {
			articles = append(articles, Article{
				PublishID:   info.MsgID,
				AppMsgID:    item.AppMsgID,
				Title:       item.Title,
				URL:         item.ContentURL,
				PublishTime: info.SentInfo.Time,
				Cover:       item.Cover,
				Digest:      item.Digest,
				ReadNum:     item.ReadNum,
				LikeNum:     item.LikeNum,
				IsDeleted:   item.IsDeleted,
			})
		}
	}

	return page, articles, nil
}
