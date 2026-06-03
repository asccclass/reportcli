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
