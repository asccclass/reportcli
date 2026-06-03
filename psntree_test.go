package main

import (
	"database/sql"
	"strings"
	"testing"
)

type execCall struct {
	query string
	args  []interface{}
}

type fakeExecer struct {
	calls []execCall
}

func (f *fakeExecer) Exec(query string, args ...interface{}) (sql.Result, error) {
	f.calls = append(f.calls, execCall{query: query, args: args})
	return nil, nil
}

func TestNewPsnTreeValidation(t *testing.T) {
	if _, err := NewPsnTree(DBConnect{}, "", []string{"role.json"}); err == nil {
		t.Fatal("NewPsnTree returned nil error for empty serverURL")
	}
	if _, err := NewPsnTree(DBConnect{}, "https://example.test/", nil); err == nil {
		t.Fatal("NewPsnTree returned nil error for empty files")
	}
}

func TestStartSyncValidation(t *testing.T) {
	tree, err := NewPsnTree(DBConnect{}, "https://example.test/", []string{"role.json"})
	if err != nil {
		t.Fatal(err)
	}

	if err := tree.StartSync("", "secret"); err == nil {
		t.Fatal("StartSync returned nil error for empty clientID")
	}
	if err := tree.StartSync("client", ""); err == nil {
		t.Fatal("StartSync returned nil error for empty clientSecret")
	}
}

func TestSyncRoleMembersDeletesAndInserts(t *testing.T) {
	execer := &fakeExecer{}
	remotePeople := []Psn{
		{UsrNo: "u1"},
		{UsrNo: "u3"},
		{UsrNo: ""},
	}
	dbPeople := []Psn{
		{UsrNo: "u1"},
		{UsrNo: "u2"},
	}

	if err := syncRoleMembers(execer, "role1", remotePeople, dbPeople, nil); err != nil {
		t.Fatalf("syncRoleMembers returned error: %v", err)
	}

	if len(execer.calls) != 2 {
		t.Fatalf("Exec called %d times, want 2: %#v", len(execer.calls), execer.calls)
	}
	if !strings.HasPrefix(execer.calls[0].query, "delete from doreuserrole") {
		t.Fatalf("first query = %q, want delete", execer.calls[0].query)
	}
	if execer.calls[0].args[0] != "u2" || execer.calls[0].args[1] != "role1" {
		t.Fatalf("delete args = %#v", execer.calls[0].args)
	}
	if !strings.HasPrefix(execer.calls[1].query, "insert into doreuserrole") {
		t.Fatalf("second query = %q, want insert", execer.calls[1].query)
	}
	if execer.calls[1].args[0] != "u3" || execer.calls[1].args[1] != "role1" {
		t.Fatalf("insert args = %#v", execer.calls[1].args)
	}
}

func TestVerifyRoleMembersPassesWhenSynced(t *testing.T) {
	remotePeople := []Psn{
		{UsrNo: "u1"},
		{UsrNo: "u2"},
		{UsrNo: ""},
	}
	dbPeople := []Psn{
		{UsrNo: "u2"},
		{UsrNo: "u1"},
	}

	if err := verifyRoleMembers("role1", remotePeople, dbPeople); err != nil {
		t.Fatalf("verifyRoleMembers returned error: %v", err)
	}
}

func TestVerifyRoleMembersReportsMissingAndUnexpected(t *testing.T) {
	remotePeople := []Psn{
		{UsrNo: "u1"},
		{UsrNo: "u3"},
	}
	dbPeople := []Psn{
		{UsrNo: "u1"},
		{UsrNo: "u2"},
	}

	err := verifyRoleMembers("role1", remotePeople, dbPeople)
	if err == nil {
		t.Fatal("verifyRoleMembers returned nil error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "missing inserts [u3]") {
		t.Fatalf("error %q does not report missing insert", msg)
	}
	if !strings.Contains(msg, "unexpected members [u2]") {
		t.Fatalf("error %q does not report unexpected member", msg)
	}
}
