package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type DBConnect struct {
	DBMSType  string `json:"DBMSType"`
	DBMS      string `json:"DBMS"`
	DbServer  string `json:"DbServer"`
	DbPort    string `json:"DbPort"`
	DbName    string `json:"DbName"`
	DbLogin   string `json:"DbLogin"`
	DbPasswd  string `json:"DbPasswd"`
	Charset   string `json:"Charset,omitempty"`
	ParseTime string `json:"ParseTime,omitempty"`
	Loc       string `json:"Loc,omitempty"`
}

type MySQL struct {
	Conn *sql.DB
}

func NewSherryDB(config DBConnect) (*MySQL, error) {
	if config.DBMS == "" {
		config.DBMS = "mysql"
	}
	if config.DBMS != "mysql" {
		return nil, fmt.Errorf("DBMS %s not supported", config.DBMS)
	}
	if config.DbServer == "" || config.DbPort == "" || config.DbName == "" || config.DbLogin == "" {
		return nil, fmt.Errorf("database config is incomplete")
	}

	dsn := mysqlDSN(config)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, err
	}

	return &MySQL{Conn: conn}, nil
}

func (db *MySQL) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.Conn.Exec(query, args...)
}

func mysqlDSN(config DBConnect) string {
	charset := config.Charset
	if charset == "" {
		charset = "utf8"
	}

	parseTime := config.ParseTime
	if parseTime == "" {
		parseTime = "True"
	}

	loc := config.Loc
	if loc == "" {
		loc = "Local"
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=%s&loc=%s",
		config.DbLogin,
		config.DbPasswd,
		config.DbServer,
		config.DbPort,
		config.DbName,
		charset,
		parseTime,
		loc,
	)
}
