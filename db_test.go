package main

import "testing"

func TestMysqlDSNDefaults(t *testing.T) {
	got := mysqlDSN(DBConnect{
		DbServer: "127.0.0.1",
		DbPort:   "3306",
		DbName:   "bpm",
		DbLogin:  "tester",
		DbPasswd: "secret",
	})

	want := "tester:secret@tcp(127.0.0.1:3306)/bpm?charset=utf8&parseTime=True&loc=Local"
	if got != want {
		t.Fatalf("mysqlDSN = %q, want %q", got, want)
	}
}

func TestMysqlDSNCustomOptions(t *testing.T) {
	got := mysqlDSN(DBConnect{
		DbServer:  "db.local",
		DbPort:    "3307",
		DbName:    "bpm",
		DbLogin:   "tester",
		DbPasswd:  "secret",
		Charset:   "utf8mb4",
		ParseTime: "true",
		Loc:       "Asia%2FTaipei",
	})

	want := "tester:secret@tcp(db.local:3307)/bpm?charset=utf8mb4&parseTime=true&loc=Asia%2FTaipei"
	if got != want {
		t.Fatalf("mysqlDSN = %q, want %q", got, want)
	}
}
