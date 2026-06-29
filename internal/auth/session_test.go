package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mudongliang/weoa-cli/internal/config"
)

func TestSessionSaveAndLoad(t *testing.T) {
	// Use a temp config dir to avoid side effects
	tmpDir := t.TempDir()
	// Override session file path by writing directly
	sessionFile := filepath.Join(tmpDir, "session.json")

	session := &Session{
		Token: "test-token-12345",
		Cookies: []Cookie{
			{Name: "test_cookie", Value: "test_value", Domain: "mp.weixin.qq.com", Path: "/", Secure: true},
			{Name: "session_id", Value: "abc123", Domain: "mp.weixin.qq.com", Path: "/", HTTPOnly: true},
		},
		UserAgent: "Mozilla/5.0 TestBrowser",
	}

	// Save directly to temp path
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	if err := os.WriteFile(sessionFile, data, 0600); err != nil {
		t.Fatalf("write session: %v", err)
	}

	// Load from temp path
	loadedData, err := os.ReadFile(sessionFile)
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	var loaded Session
	if err := json.Unmarshal(loadedData, &loaded); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}

	if loaded.Token != session.Token {
		t.Errorf("token mismatch: got %q, want %q", loaded.Token, session.Token)
	}
	if len(loaded.Cookies) != len(session.Cookies) {
		t.Fatalf("cookie count mismatch: got %d, want %d", len(loaded.Cookies), len(session.Cookies))
	}
	if loaded.Cookies[0].Name != "test_cookie" {
		t.Errorf("cookie name mismatch: got %q, want 'test_cookie'", loaded.Cookies[0].Name)
	}
	if loaded.UserAgent != session.UserAgent {
		t.Errorf("user agent mismatch: got %q, want %q", loaded.UserAgent, session.UserAgent)
	}
}

func TestCookieToHTTPCookie(t *testing.T) {
	c := Cookie{
		Name:     "session",
		Value:    "xyz789",
		Domain:   ".mp.weixin.qq.com",
		Path:     "/cgi-bin",
		Secure:   true,
		HTTPOnly: true,
	}

	hc := c.ToHTTPCookie()
	if hc.Name != c.Name {
		t.Errorf("Name: got %q, want %q", hc.Name, c.Name)
	}
	if hc.Value != c.Value {
		t.Errorf("Value: got %q, want %q", hc.Value, c.Value)
	}
	if hc.Domain != c.Domain {
		t.Errorf("Domain: got %q, want %q", hc.Domain, c.Domain)
	}
	if hc.Path != c.Path {
		t.Errorf("Path: got %q, want %q", hc.Path, c.Path)
	}
	if hc.Secure != c.Secure {
		t.Errorf("Secure: got %v, want %v", hc.Secure, c.Secure)
	}
	if hc.HttpOnly != c.HTTPOnly {
		t.Errorf("HttpOnly: got %v, want %v", hc.HttpOnly, c.HTTPOnly)
	}
}

func TestSessionHTTPCookies(t *testing.T) {
	s := &Session{
		Token: "test",
		Cookies: []Cookie{
			{Name: "a", Value: "1"},
			{Name: "b", Value: "2"},
			{Name: "c", Value: "3"},
		},
	}

	httpCookies := s.HTTPCookies()
	if len(httpCookies) != 3 {
		t.Fatalf("expected 3 cookies, got %d", len(httpCookies))
	}
	if httpCookies[1].Name != "b" {
		t.Errorf("expected cookie name 'b', got %q", httpCookies[1].Name)
	}
}

func TestSessionClear(t *testing.T) {
	// Save a session first, then clear it
	if err := config.EnsureDir(); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}

	s := &Session{Token: "will-be-deleted"}
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	if !Exists() {
		t.Fatal("session should exist after save")
	}

	if err := Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}

	if Exists() {
		t.Error("session should not exist after clear")
	}
}
