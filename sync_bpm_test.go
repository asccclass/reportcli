package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitSyncFiles(t *testing.T) {
	got := splitSyncFiles("role-a.json, role-b.json,, role-c.json ")
	want := []string{"role-a.json", "role-b.json", "role-c.json"}

	if len(got) != len(want) {
		t.Fatalf("splitSyncFiles length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitSyncFiles[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := `{
		"Title": "BPM sync",
		"WebServiceUrl": "https://example.test/",
		"Dbconnect": {
			"DBMS": "mysql",
			"DbServer": "127.0.0.1",
			"DbPort": "3306",
			"DbName": "bpm",
			"DbLogin": "tester",
			"DbPasswd": "secret"
		},
		"orgTreeServerUrl": "https://org.example.test/",
		"syncFiles": "role-a.json,role-b.json",
		"clientID": "client",
		"clientSecret": "secret"
	}`

	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}
	if config.Title != "BPM sync" {
		t.Fatalf("Title = %q, want BPM sync", config.Title)
	}
	if config.Dbconnect.DbServer != "127.0.0.1" {
		t.Fatalf("DbServer = %q, want 127.0.0.1", config.Dbconnect.DbServer)
	}
	if config.SyncFiles != "role-a.json,role-b.json" {
		t.Fatalf("SyncFiles = %q", config.SyncFiles)
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"Title":`), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := loadConfig(path); err == nil {
		t.Fatal("loadConfig returned nil error for invalid JSON")
	}
}

func TestConfigureProxy(t *testing.T) {
	envNames := []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"}
	for _, name := range envNames {
		t.Setenv(name, "")
	}

	if err := configureProxy(" "); err != nil {
		t.Fatalf("configureProxy empty returned error: %v", err)
	}
	for _, name := range envNames {
		if got := os.Getenv(name); got != "" {
			t.Fatalf("%s = %q, want empty", name, got)
		}
	}

	proxy := "http://127.0.0.1:8080"
	if err := configureProxy(proxy); err != nil {
		t.Fatalf("configureProxy returned error: %v", err)
	}
	for _, name := range envNames {
		if got := os.Getenv(name); got != proxy {
			t.Fatalf("%s = %q, want %q", name, got, proxy)
		}
	}

	if err := configureProxy("127.0.0.1:8080"); err == nil {
		t.Fatal("configureProxy returned nil error for proxy without scheme")
	}
}
