package client

import (
	"fmt"
	"net/http/cookiejar"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/mudongliang/weoa-cli/internal/auth"
)

const BaseURL = "https://mp.weixin.qq.com"

// Client wraps resty with WeChat backend session management.
type Client struct {
	resty  *resty.Client
	session *auth.Session
}

// New creates a new Client from a saved session.
func New(session *auth.Session) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	baseURL, _ := url.Parse(BaseURL)
	jar.SetCookies(baseURL, session.HTTPCookies())

	r := resty.New().
		SetBaseURL(BaseURL).
		SetCookieJar(jar).
		SetHeader("User-Agent", session.UserAgent).
		SetHeader("Accept", "application/json, text/plain, */*").
		SetHeader("Accept-Language", "zh-CN,zh;q=0.9").
		SetHeader("Referer", BaseURL+"/cgi-bin/home").
		SetDebug(false)

	return &Client{resty: r, session: session}, nil
}

// Session returns the underlying session.
func (c *Client) Session() *auth.Session {
	return c.session
}

// R returns the underlying resty Request builder, with token and lang
// query params pre-set.
func (c *Client) R() *resty.Request {
	return c.resty.R().
		SetQueryParam("token", c.session.Token).
		SetQueryParam("lang", "zh_CN")
}

// Get is a convenience method for GET requests that injects token/lang.
func (c *Client) Get(path string) (*resty.Response, error) {
	return c.R().Get(path)
}

// GetWithParams is a convenience method for GET requests with extra query params.
func (c *Client) GetWithParams(path string, params map[string]string) (*resty.Response, error) {
	req := c.R()
	for k, v := range params {
		req.SetQueryParam(k, v)
	}
	return req.Get(path)
}

// JSONBody is a helper to set JSON content-type and body.
func (c *Client) JSONBody(body interface{}) *resty.Request {
	return c.R().SetHeader("Content-Type", "application/json").SetBody(body)
}

// Token returns the current session token.
func (c *Client) Token() string {
	return c.session.Token
}
