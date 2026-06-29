package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"
)

const (
	loginURL       = "https://mp.weixin.qq.com/"
	homePath       = "/cgi-bin/home"
	loginTimeout   = 120 * time.Second
	postLoginPause = 2 * time.Second
)

// LoginResult holds the outcome of a successful login.
type LoginResult struct {
	AccountName string
	Session     *Session
}

// Login opens a browser for QR-code scanning and returns a saved session.
// It blocks until the user scans the QR code with WeChat or the timeout expires.
func Login(ctx context.Context) (*LoginResult, error) {
	// Auto-install Playwright browsers if not already present (idempotent).
	if err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
	}); err != nil {
		return nil, fmt.Errorf("install playwright browser: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("launch playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}
	defer browser.Close()

	page, err := browser.NewPage(playwright.BrowserNewPageOptions{
		UserAgent: playwright.String(
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		),
	})
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}

	fmt.Println("Opening mp.weixin.qq.com login page...")
	fmt.Println("Please scan the QR code with WeChat to log in.")

	if _, err := page.Goto(loginURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return nil, fmt.Errorf("navigate to login: %w", err)
	}

	// Wait for navigation to the dashboard (home page), indicating successful login.
	loginCtx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()

	err = waitForLogin(loginCtx, page)
	if err != nil {
		return nil, fmt.Errorf("login timeout: %w", err)
	}

	// Give the page a moment to fully settle
	time.Sleep(postLoginPause)

	// Extract token from current URL
	currentURL := page.URL()
	token := extractToken(currentURL)
	if token == "" {
		return nil, fmt.Errorf("could not extract token from URL: %s", currentURL)
	}

	// Get account name from page title
	title, err := page.Title()
	if err != nil {
		title = "Unknown"
	}
	accountName := strings.TrimSuffix(title, " - 微信公众平台")
	accountName = strings.TrimSpace(accountName)

	// Get cookies
	pwCookies, err := page.Context().Cookies()
	if err != nil {
		return nil, fmt.Errorf("get cookies: %w", err)
	}

	// Get User-Agent
	ua, err := page.Evaluate("navigator.userAgent")
	if err != nil {
		ua = ""
	}
	uaStr, _ := ua.(string)

	// Convert Playwright cookies to our serializable Cookie type
	cookies := make([]Cookie, 0, len(pwCookies))
	for _, c := range pwCookies {
		cookies = append(cookies, Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  float64(c.Expires),
			Secure:   c.Secure,
			HTTPOnly: c.HttpOnly,
		})
	}

	session := &Session{
		Token:     token,
		Cookies:   cookies,
		UserAgent: uaStr,
	}

	if err := session.Save(); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	return &LoginResult{
		AccountName: accountName,
		Session:     session,
	}, nil
}

// waitForLogin polls until the page navigates to the home (dashboard) URL
// or the context is cancelled.
func waitForLogin(ctx context.Context, page playwright.Page) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			u := page.URL()
			if strings.Contains(u, homePath) {
				return nil
			}
			// Also check if we're on any post-login page
			if strings.Contains(u, "mp.weixin.qq.com") && !strings.Contains(u, "login") && !strings.Contains(u, "bizlogin") {
				if strings.Contains(u, "token=") {
					return nil
				}
			}
		}
	}
}

// extractToken pulls the token value from a WeChat backend URL.
func extractToken(rawURL string) string {
	idx := strings.Index(rawURL, "token=")
	if idx == -1 {
		return ""
	}
	rest := rawURL[idx+6:]
	end := strings.Index(rest, "&")
	if end == -1 {
		return rest
	}
	return rest[:end]
}
