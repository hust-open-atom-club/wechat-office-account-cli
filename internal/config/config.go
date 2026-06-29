package config

import (
	"os"
	"path/filepath"
)

const (
	AppName   = "weoa-cli"
	EnvPrefix = "WEOA"
)

// DefaultDir returns the default config directory: ~/.config/weoa-cli
func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "."+AppName)
	}
	return filepath.Join(home, ".config", AppName)
}

// SessionFile returns the path to the session JSON file.
func SessionFile() string {
	return filepath.Join(DefaultDir(), "session.json")
}

// DBFile returns the path to the SQLite database file.
func DBFile() string {
	return filepath.Join(DefaultDir(), "cache.db")
}

// EnsureDir creates the config directory if it doesn't exist.
func EnsureDir() error {
	return os.MkdirAll(DefaultDir(), 0700)
}
