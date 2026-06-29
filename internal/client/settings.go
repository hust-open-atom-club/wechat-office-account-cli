package client

import (
	"encoding/json"
	"fmt"

	"github.com/mudongliang/weoa-cli/internal/auth"
)

// AccountInfo holds the parsed account profile from the settings page API.
type AccountInfo struct {
	Nickname           string   `json:"nickname"`            // 公众号名称
	WeChatID           string   `json:"wechat_id"`           // 微信号
	OriginalID         string   `json:"original_id"`         // 原始ID
	Signature          string   `json:"signature"`           // 简介
	Email              string   `json:"email"`               // 绑定邮箱
	FinderNickname     string   `json:"finder_nickname"`     // 视频号
	WxNickname         string   `json:"wx_nickname"`         // 管理员微信昵称
	Categories         []string `json:"categories"`          // 分类
	FansCount          int      `json:"fans_count"`          // 粉丝数
	Location           string   `json:"location"`            // 所在地
	Verified           bool     `json:"verified"`            // 是否认证
	Searchable         bool     `json:"searchable"`          // 是否允许搜索
	Token              string   `json:"token"`               // 当前 token（脱敏）
}

// FetchAccountInfo fetches the account profile from the settings API.
func FetchAccountInfo(session *auth.Session) (*AccountInfo, error) {
	c, err := New(session)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	resp, err := c.GetWithParams("/cgi-bin/settingpage", map[string]string{
		"t":      "setting/index",
		"action": "index",
		"f":      "json",
	})
	if err != nil {
		return nil, fmt.Errorf("request settings: %w", err)
	}

	var result settingResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}

	info := result.SettingInfo

	// Extract categories
	categories := make([]string, 0)
	for _, cat := range info.BizCategoryInfo.BizCategories {
		name := cat.FirstCateName
		if cat.SecondCateName != "" && cat.SecondCateName != "公众号(订阅号)类目根节点" {
			name += " / " + cat.SecondCateName
		}
		categories = append(categories, name)
	}

	return &AccountInfo{
		Nickname:       info.Nickname.Nickname,
		WeChatID:       info.Username,
		OriginalID:     info.OriginalUsername,
		Signature:      info.Intro.Signature,
		Email:          info.BindEmail.Account,
		FinderNickname: info.BindFinderStatus.FinderNickname,
		WxNickname:     info.BindFinderStatus.WxNickname,
		Categories:     categories,
		FansCount:      info.TotalFansNum,
		Location:       info.LocationInfo.Position,
		Verified:       info.WxVerify.WxVerifyStatus != 0,
		Searchable:     info.SearchOpen == 1,
		Token:          session.Token,
	}, nil
}

// --- Raw API response types ---

type settingResponse struct {
	BaseResp    baseResp    `json:"base_resp"`
	SettingInfo settingInfo `json:"setting_info"`
}

type baseResp struct {
	Ret    int    `json:"ret"`
	ErrMsg string `json:"err_msg"`
}

type settingInfo struct {
	Nickname           nicknameInfo      `json:"nickname"`
	Username           string            `json:"username"`
	OriginalUsername   string            `json:"original_username"`
	Intro              introInfo         `json:"intro"`
	BindEmail          bindEmailInfo     `json:"bind_email"`
	BindFinderStatus   bindFinderInfo    `json:"bind_finder_status"`
	BizCategoryInfo    bizCategoryInfo   `json:"biz_category_info"`
	TotalFansNum       int               `json:"total_fans_num"`
	LocationInfo       locationInfo      `json:"location_info"`
	WxVerify           wxVerifyInfo      `json:"wxverify"`
	SearchOpen         int               `json:"search_open"`
}

type nicknameInfo struct {
	Nickname string `json:"nickname"`
}

type introInfo struct {
	Signature string `json:"signature"`
}

type bindEmailInfo struct {
	Account string `json:"account"`
}

type bindFinderInfo struct {
	FinderNickname string `json:"finder_nickname"`
	WxNickname     string `json:"wx_nickname"`
}

type bizCategoryInfo struct {
	BizCategories []bizCategory `json:"biz_categories"`
}

type bizCategory struct {
	FirstCateName  string `json:"first_cate_name"`
	SecondCateName string `json:"second_cate_name"`
}

type locationInfo struct {
	Position string `json:"position"`
}

type wxVerifyInfo struct {
	WxVerifyStatus int `json:"wxverify_status"`
}
