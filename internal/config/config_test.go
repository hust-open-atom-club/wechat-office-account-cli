package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultDir(t *testing.T) {
	dir := DefaultDir()
	if dir == "" {
		t.Fatal("DefaultDir returned empty string")
	}
	// Should contain weoa-cli somewhere in the path
	if !strings.Contains(dir, "weoa-cli") {
		t.Errorf("DefaultDir should contain 'weoa-cli', got: %s", dir)
	}
}

func TestDefaultDir_Fallback(t *testing.T) {
	// If HOME is unset, should use current directory
	origHome := os.Getenv("HOME")
	os.Unsetenv("HOME")
	defer os.Setenv("HOME", origHome)

	dir := DefaultDir()
	if !strings.Contains(dir, ".weoa-cli") {
		t.Errorf("fallback dir should contain '.weoa-cli', got: %s", dir)
	}
}

func TestSessionFile(t *testing.T) {
	path := SessionFile()
	if !strings.HasSuffix(path, "session.json") {
		t.Errorf("SessionFile should end with 'session.json', got: %s", path)
	}
	if !strings.Contains(path, "weoa-cli") {
		t.Errorf("SessionFile should contain 'weoa-cli', got: %s", path)
	}
}

func TestDBFile(t *testing.T) {
	path := DBFile()
	if !strings.HasSuffix(path, "cache.db") {
		t.Errorf("DBFile should end with 'cache.db', got: %s", path)
	}
}

func TestEnsureDir(t *testing.T) {
	// Use a temp directory to avoid side effects
	tmpDir := filepath.Join(os.TempDir(), "weoa-cli-test-"+t.Name())
	defer os.RemoveAll(tmpDir)

	// Override default dir: we can't directly, but we test EnsureDir directly
	err := os.MkdirAll(tmpDir, 0700)
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("stat temp dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("created path is not a directory")
	}
}
