package auth

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/mudongliang/weoa-cli/internal/config"
)

// Cookie is a JSON-serializable http.Cookie representation.
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	Secure   bool    `json:"secure"`
	HTTPOnly bool    `json:"http_only"`
}

// ToHTTPCookie converts to a standard http.Cookie.
func (c *Cookie) ToHTTPCookie() *http.Cookie {
	return &http.Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HttpOnly: c.HTTPOnly,
	}
}

// Session holds the persisted authentication state.
type Session struct {
	Token     string   `json:"token"`
	Cookies   []Cookie `json:"cookies"`
	UserAgent string   `json:"user_agent"`
}

// Load reads the session from disk.
func Load() (*Session, error) {
	data, err := os.ReadFile(config.SessionFile())
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save writes the session to disk.
func (s *Session) Save() error {
	if err := config.EnsureDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(config.SessionFile(), data, 0600)
}

// Clear removes the session file.
func Clear() error {
	return os.Remove(config.SessionFile())
}

// Exists returns true if a saved session file exists.
func Exists() bool {
	_, err := os.Stat(config.SessionFile())
	return err == nil
}

// HTTPCookies converts the session cookies to standard http.Cookie slice.
func (s *Session) HTTPCookies() []*http.Cookie {
	result := make([]*http.Cookie, len(s.Cookies))
	for i := range s.Cookies {
		result[i] = s.Cookies[i].ToHTTPCookie()
	}
	return result
}
