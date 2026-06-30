package auth

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
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

// LoginOptions controls the login UI.
type LoginOptions struct {
	PrintQR bool
}

// LoginResult holds the outcome of a successful login.
type LoginResult struct {
	AccountName string
	Session     *Session
}

// Login opens a browser for QR-code scanning and returns a saved session.
// It blocks until the user scans the QR code with WeChat or the timeout expires.
func Login(ctx context.Context, options ...LoginOptions) (*LoginResult, error) {
	opts := LoginOptions{}
	if len(options) > 0 {
		opts = options[0]
	}

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
		Headless: playwright.Bool(opts.PrintQR),
	})
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}
	defer browser.Close()

	page, err := browser.NewPage(playwright.BrowserNewPageOptions{
		UserAgent: playwright.String(
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		),
		Viewport: &playwright.Size{Width: 1280, Height: 900},
	})
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}

	if opts.PrintQR {
		fmt.Println("Loading mp.weixin.qq.com login page in headless mode...")
	} else {
		fmt.Println("Opening mp.weixin.qq.com login page...")
		fmt.Println("Please scan the QR code with WeChat to log in.")
	}

	if _, err := page.Goto(loginURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return nil, fmt.Errorf("navigate to login: %w", err)
	}

	if opts.PrintQR {
		if err := printLoginQRCode(page); err != nil {
			return nil, err
		}
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

func printLoginQRCode(page playwright.Page) error {
	pngBytes, err := captureLoginQRCode(page)
	if err != nil {
		return err
	}

	img, _, err := image.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return fmt.Errorf("decode QR code image: %w", err)
	}

	fmt.Println()
	fmt.Print(renderQRImage(img, 48))
	fmt.Println()
	fmt.Println("Please scan the QR code with WeChat to log in.")
	return nil
}

func captureLoginQRCode(page playwright.Page) ([]byte, error) {
	selectors := []string{
		"img[src*='qrcode']:visible",
		"img[src*='scanloginqrcode']:visible",
		".login__type__container__scan__qrcode img:visible",
		".qrcode img:visible",
		"canvas:visible",
	}

	for _, selector := range selectors {
		locator := page.Locator(selector).First()
		err := locator.WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(5000),
		})
		if err != nil {
			continue
		}
		pngBytes, err := locator.Screenshot(playwright.LocatorScreenshotOptions{
			Scale: playwright.ScreenshotScaleCss,
		})
		if err == nil && len(pngBytes) > 0 {
			return pngBytes, nil
		}
	}

	if _, err := page.Evaluate(`() => {
		const candidates = Array.from(document.querySelectorAll('img, canvas'))
			.map((el) => {
				const rect = el.getBoundingClientRect();
				const style = window.getComputedStyle(el);
				const visible = rect.width >= 120 && rect.height >= 120 &&
					style.visibility !== 'hidden' && style.display !== 'none';
				const ratio = rect.width / rect.height;
				const squareish = ratio > 0.75 && ratio < 1.35;
				return { el, visible, squareish, area: rect.width * rect.height };
			})
			.filter((item) => item.visible && item.squareish)
			.sort((a, b) => b.area - a.area);
		if (candidates.length === 0) {
			return false;
		}
		candidates[0].el.setAttribute('data-weoa-login-qr', '1');
		return true;
	}`); err == nil {
		locator := page.Locator("[data-weoa-login-qr='1']").First()
		pngBytes, err := locator.Screenshot(playwright.LocatorScreenshotOptions{
			Scale: playwright.ScreenshotScaleCss,
		})
		if err == nil && len(pngBytes) > 0 {
			return pngBytes, nil
		}
	}

	return nil, fmt.Errorf("find login QR code on page")
}

func renderQRImage(img image.Image, targetWidth int) string {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return ""
	}
	if targetWidth <= 0 || targetWidth > width {
		targetWidth = width
	}
	targetHeight := height * targetWidth / width
	if targetHeight <= 0 {
		targetHeight = 1
	}

	var b strings.Builder
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			if sampleDark(img, bounds, x, y, targetWidth, targetHeight) {
				b.WriteString("\x1b[40m  ")
			} else {
				b.WriteString("\x1b[47m  ")
			}
		}
		b.WriteString("\x1b[0m\n")
	}
	return b.String()
}

func sampleDark(img image.Image, bounds image.Rectangle, x, y, targetWidth, targetHeight int) bool {
	startX := bounds.Min.X + x*bounds.Dx()/targetWidth
	endX := bounds.Min.X + (x+1)*bounds.Dx()/targetWidth
	startY := bounds.Min.Y + y*bounds.Dy()/targetHeight
	endY := bounds.Min.Y + (y+1)*bounds.Dy()/targetHeight
	if endX <= startX {
		endX = startX + 1
	}
	if endY <= startY {
		endY = startY + 1
	}

	var total uint64
	var count uint64
	for py := startY; py < endY; py++ {
		for px := startX; px < endX; px++ {
			r, g, b, a := img.At(px, py).RGBA()
			if a < 0x8000 {
				continue
			}
			total += uint64(luminance(color.RGBA64{
				R: uint16(r),
				G: uint16(g),
				B: uint16(b),
				A: uint16(a),
			}))
			count++
		}
	}
	if count == 0 {
		return false
	}
	return total/count < 0x8000
}

func luminance(c color.RGBA64) uint32 {
	return (uint32(c.R)*299 + uint32(c.G)*587 + uint32(c.B)*114) / 1000
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
