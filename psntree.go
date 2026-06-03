package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"

	SherryClient "reportcli/client"
)

type PsnTree struct {
	ServerURL string
	Files     []string
	Bearer    string
	DBConfig  DBConnect
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

	fmt.Printf("sync remote file %s...\n", fileName)
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

	if err := fillUserNumbers(conn.Conn, conn, people); err != nil {
		return err
	}

	peopleInDB, err := listPeopleInRole(conn.Conn, roleID)
	if err != nil {
		return err
	}

	if err := syncRoleMembers(conn, roleID, people, peopleInDB); err != nil {
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

func fillUserNumbers(db *sql.DB, execer dbExecer, people []Psn) error {
	for i, value := range people {
		usrNo, ssoID, dep, depID, err := findUserBySSOID(db, value.SysID)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
			var depIDOut string
			usrNo, ssoID, dep, depIDOut, err = findUserByNameAndDepID(db, value.Name, value.DepID)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == nil {
				depID = depIDOut
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
				if _, err := execer.Exec("update doreuser set ssoID=?,department=?,depID=? where usrNo=?", valSysID, valDepName, valDepID, usrNo); err != nil {
					return err
				}
				fmt.Printf("update ssoID: %v\n", value)
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

func findUserByNameAndDepID(db *sql.DB, name, depID string) (usrNo, ssoID, dep, depIDOut string, err error) {
	row := db.QueryRow("select usrNo,ssoID,department,depID from doreuser where name=? and depID=?", name, depID)
	err = row.Scan(&usrNo, &ssoID, &dep, &depIDOut)
	return usrNo, ssoID, dep, depIDOut, err
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

func syncRoleMembers(execer dbExecer, roleID string, remotePeople, dbPeople []Psn) error {
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
		fmt.Printf("   delete from doreuserrole where usrNo=%s and roleID=%s\n", person.UsrNo, roleID)
	}

	for _, person := range remotePeople {
		if person.UsrNo == "" || person.Op != "" {
			continue
		}

		if _, err := execer.Exec("insert into doreuserrole(usrNo, roleID) values(?, ?)", person.UsrNo, roleID); err != nil {
			return err
		}
		fmt.Printf("   insert into doreuserrole(usrNo, roleID) values(%s, %s)\n", person.UsrNo, roleID)
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
