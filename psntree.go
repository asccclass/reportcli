package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	SherryClient "reportcli/client"
)

type PsnTree struct {
	ServerURL  string
	Files      []string
	Bearer     string
	DBConfig   DBConnect
	ReportLogs []string
}

type Psn struct {
	DepID   string `json:"depID"`
	DepName string `json:"depName"`
	Typez   string `json:"typez"`
	SysID   string `json:"SysId"`
	Name    string `json:"Name"`
	UsrNo   string `json:"usrNo"`
	Op      string `json:"op"`
	InDate  string `json:"inDate"`
}

type dbExecer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func NewPsnTree(config DBConnect, serverURL string, files []string) (*PsnTree, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("ServerUrl is empty")
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("files is empty, please check syncFiles")
	}

	return &PsnTree{
		ServerURL: serverURL,
		Files:     files,
		DBConfig:  config,
	}, nil
}

func (app *PsnTree) AddFiles(fileName string) {
	if fileName != "" {
		app.Files = append(app.Files, fileName)
	}
}

func (app *PsnTree) StartSync(clientID, clientSecret string) error {
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("client ID or client Secret is empty")
	}

	payload := &struct {
		ClientID     string `json:"clientID"`
		ClientSecret string `json:"clientSecret"`
	}{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal access token payload: %w", err)
	}

	return app.DoCompare(string(data))
}

func (app *PsnTree) DoCompare(payload string) error {
	client, err := SherryClient.NewClient("GET", app.ServerURL+"accesstoken", payload)
	if err != nil {
		return fmt.Errorf("create access token client: %w", err)
	}

	result, err := client.Do()
	if err != nil {
		return err
	}
	if result == "" {
		return fmt.Errorf("get access token from %s returned empty string", app.ServerURL+"accesstoken")
	}

	errMsg := &struct {
		Msg string `json:"errMsg"`
	}{}
	if err := json.Unmarshal([]byte(result), errMsg); err == nil && errMsg.Msg != "" {
		return fmt.Errorf("get access token error: %s", errMsg.Msg)
	}

	ber := &SherryClient.BearerToken{}
	if err := json.Unmarshal([]byte(result), ber); err != nil {
		return fmt.Errorf("parse access token response %q: %w", result, err)
	}
	app.Bearer = ber.Token

	for _, fileName := range app.Files {
		if err := app.GetRemoteFileAndCompare(client, fileName); err != nil {
			return err
		}
	}

	return nil
}

func (app *PsnTree) GetRemoteFileAndCompare(client *SherryClient.SryClient, fileName string) error {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	_ = writer.WriteField("user", "bpm")
	_ = writer.WriteField("action", "GET")
	_ = writer.WriteField("file", fileName)
	_ = writer.WriteField("object", "/read/"+fileName)
	_ = writer.WriteField("systemName", "opendatacenter")
	if err := writer.Close(); err != nil {
		return err
	}

	msg := fmt.Sprintf("sync remote file %s...", fileName)
	fmt.Println(msg)
	app.log(msg)
	req, err := http.NewRequest(http.MethodGet, app.ServerURL+"read/"+fileName, payload)
	if err != nil {
		return err
	}
	client.Request = req
	client.AddHeader("Authorization", "Bearer "+app.Bearer)
	client.AddHeader("Content-Type", writer.FormDataContentType())

	result, err := client.Do()
	if err != nil {
		return err
	}
	if result == "" {
		return fmt.Errorf("remote file %s result is empty", fileName)
	}

	if err := os.WriteFile(fileName, []byte(result), 0644); err != nil {
		return fmt.Errorf("save remote file %s: %w", fileName, err)
	}

	people := []Psn{}
	if err := json.Unmarshal([]byte(result), &people); err != nil {
		return fmt.Errorf("parse remote file %s: %w", fileName, err)
	}
	if len(people) == 0 {
		fmt.Printf("remote file %s has no people.\n", fileName)
		return nil
	}

	conn, err := NewSherryDB(app.DBConfig)
	if err != nil {
		return err
	}
	defer conn.Conn.Close()

	roleID, err := lookupRoleID(conn.Conn, people[0].Typez)
	if err != nil {
		return err
	}

	if err := fillUserNumbers(conn.Conn, conn, people, app.log); err != nil {
		return err
	}

	peopleInDB, err := listPeopleInRole(conn.Conn, roleID)
	if err != nil {
		return err
	}

	if err := syncRoleMembers(conn, roleID, people, peopleInDB, app.log); err != nil {
		return err
	}

	peopleAfterSync, err := listPeopleInRole(conn.Conn, roleID)
	if err != nil {
		return err
	}
	if err := verifyRoleMembers(roleID, people, peopleAfterSync); err != nil {
		return err
	}
	fmt.Printf("   verified roleID=%s members after sync\n", roleID)

	return nil
}

func lookupRoleID(db *sql.DB, typez string) (string, error) {
	if typez == "" {
		return "", fmt.Errorf("typez is empty")
	}

	roleID := ""
	row := db.QueryRow("select roleID from role where namez=?", typez)
	if err := row.Scan(&roleID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("role not found for typez %s", typez)
		}
		return "", err
	}

	return roleID, nil
}

func fillUserNumbers(db *sql.DB, execer dbExecer, people []Psn, logFunc func(string)) error {
	for i, value := range people {
		usrNo, ssoID, dep, depID, err := findUserBySSOID(db, value.SysID)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
			usrNo, ssoID, dep, depID, err = findUserByName(db, value.Name)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
		}

		if usrNo != "" {
			dbSSOID := strings.TrimSpace(ssoID)
			dbDep := strings.TrimSpace(dep)
			dbDepID := strings.TrimSpace(depID)
			valSysID := strings.TrimSpace(value.SysID)
			valDepName := strings.TrimSpace(value.DepName)
			valDepID := strings.TrimSpace(value.DepID)

			if dbSSOID == "" || dbDep != valDepName || dbDepID != valDepID || dbSSOID != valSysID {
				fmt.Printf("DEBUG: usrNo=%q, dbSSOID=%q (len=%d), dbDep=%q (len=%d), dbDepID=%q (len=%d) vs valSysID=%q (len=%d), valDepName=%q (len=%d), valDepID=%q (len=%d)\n",
					usrNo, dbSSOID, len(dbSSOID), dbDep, len(dbDep), dbDepID, len(dbDepID),
					valSysID, len(valSysID), valDepName, len(valDepName), valDepID, len(valDepID))
				query := "update doreuser set ssoID=?,department=?,depID=? where usrNo=?"
				if _, err := execer.Exec(query, valSysID, valDepName, valDepID, usrNo); err != nil {
					return err
				}
				msg := fmt.Sprintf("更新使用者資訊 [usrNo: %s, 姓名: %s] 到資料表 doreuser:\n  - 舊資料: ssoID=%q, 單位=%q (代碼: %q)\n  - 新資料: ssoID=%q, 單位=%q (代碼: %q)\n  - 執行 SQL: %s (參數: ssoID=%q, department=%q, depID=%q, usrNo=%q)",
					usrNo, value.Name, dbSSOID, dbDep, dbDepID, valSysID, valDepName, valDepID, query, valSysID, valDepName, valDepID, usrNo)
				fmt.Println(msg)
				if logFunc != nil {
					logFunc(msg)
				}
			}
		}

		people[i].UsrNo = usrNo
	}

	return nil
}

func findUserBySSOID(db *sql.DB, sysID string) (usrNo, ssoID, dep, depID string, err error) {
	row := db.QueryRow("select usrNo,ssoID,department,depID from doreuser where ssoID=?", sysID)
	err = row.Scan(&usrNo, &ssoID, &dep, &depID)
	return usrNo, ssoID, dep, depID, err
}

func findUserByName(db *sql.DB, name string) (usrNo, ssoID, dep, depID string, err error) {
	rows, err := db.Query("select usrNo,ssoID,department,depID from doreuser where name=?", name)
	if err != nil {
		return "", "", "", "", err
	}
	defer rows.Close()

	type User struct {
		usrNo, ssoID, dep, depID string
	}
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.usrNo, &u.ssoID, &u.dep, &u.depID); err == nil {
			users = append(users, u)
		}
	}
	if len(users) == 0 {
		return "", "", "", "", sql.ErrNoRows
	}
	if len(users) == 1 {
		return users[0].usrNo, users[0].ssoID, users[0].dep, users[0].depID, nil
	}
	for _, u := range users {
		if u.ssoID == "" {
			return u.usrNo, u.ssoID, u.dep, u.depID, nil
		}
	}
	return users[0].usrNo, users[0].ssoID, users[0].dep, users[0].depID, nil
}

func listPeopleInRole(db *sql.DB, roleID string) ([]Psn, error) {
	sqlstr := "select c.usrNo,c.depID,c.department,c.ssoID,c.name from doreuserrole b, doreuser c where b.usrNo=c.usrNo and b.roleID=?"
	rows, err := db.Query(sqlstr, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	people := []Psn{}
	for rows.Next() {
		p := Psn{}
		if err := rows.Scan(&p.UsrNo, &p.DepID, &p.DepName, &p.SysID, &p.Name); err != nil {
			return nil, err
		}
		people = append(people, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return people, nil
}

func syncRoleMembers(execer dbExecer, roleID string, remotePeople, dbPeople []Psn, logFunc func(string)) error {
	remoteByUsrNo := make(map[string]int, len(remotePeople))
	for i, person := range remotePeople {
		if person.UsrNo != "" {
			remoteByUsrNo[person.UsrNo] = i
		}
	}

	for _, person := range dbPeople {
		index, ok := remoteByUsrNo[person.UsrNo]
		if ok {
			remotePeople[index].Op = "exists"
			continue
		}

		if _, err := execer.Exec("delete from doreuserrole where usrNo=? and roleID=?", person.UsrNo, roleID); err != nil {
			return err
		}
		msg := fmt.Sprintf("   delete from doreuserrole where usrNo=%s and roleID=%s", person.UsrNo, roleID)
		fmt.Println(msg)
		if logFunc != nil {
			logFunc(msg)
		}
	}

	for _, person := range remotePeople {
		if person.UsrNo == "" || person.Op != "" {
			continue
		}

		if _, err := execer.Exec("insert into doreuserrole(usrNo, roleID) values(?, ?)", person.UsrNo, roleID); err != nil {
			return err
		}
		msg := fmt.Sprintf("   insert into doreuserrole(usrNo, roleID) values(%s, %s)", person.UsrNo, roleID)
		fmt.Println(msg)
		if logFunc != nil {
			logFunc(msg)
		}
	}

	return nil
}

func verifyRoleMembers(roleID string, remotePeople, dbPeople []Psn) error {
	expected := make(map[string]bool, len(remotePeople))
	for _, person := range remotePeople {
		if person.UsrNo != "" {
			expected[person.UsrNo] = true
		}
	}

	actual := make(map[string]bool, len(dbPeople))
	for _, person := range dbPeople {
		if person.UsrNo != "" {
			actual[person.UsrNo] = true
		}
	}

	missing := []string{}
	for usrNo := range expected {
		if !actual[usrNo] {
			missing = append(missing, usrNo)
		}
	}

	unexpected := []string{}
	for usrNo := range actual {
		if !expected[usrNo] {
			unexpected = append(unexpected, usrNo)
		}
	}

	if len(missing) == 0 && len(unexpected) == 0 {
		return nil
	}

	sort.Strings(missing)
	sort.Strings(unexpected)
	return fmt.Errorf(
		"verify role members failed for roleID %s: missing inserts [%s], unexpected members [%s]",
		roleID,
		strings.Join(missing, ", "),
		strings.Join(unexpected, ", "),
	)
}

func (app *PsnTree) log(msg string) {
	app.ReportLogs = append(app.ReportLogs, msg)
}

func (app *PsnTree) GenerateReport() error {
	today := time.Now().Format("20060102")
	fileName := fmt.Sprintf("report_%s.txt", today)

	var buf strings.Builder
	buf.WriteString("========================================\n")
	buf.WriteString(" 全院同仁學習時數管理系統每日同步報告\n")
	buf.WriteString(fmt.Sprintf(" 執行時間: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	buf.WriteString("========================================\n\n")

	if len(app.ReportLogs) == 0 {
		buf.WriteString("沒有任何資料變更。\n")
	} else {
		for _, log := range app.ReportLogs {
			buf.WriteString(log)
			buf.WriteString("\n")
		}
	}

	err := os.WriteFile(fileName, []byte(buf.String()), 0644)
	if err != nil {
		return fmt.Errorf("write report file: %w", err)
	}

	fmt.Printf("Daily report generated: %s\n", fileName)
	return nil
}
