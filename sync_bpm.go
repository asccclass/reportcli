package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	SherryErrorExecuter "reportcli/errorexecuter"
)

type System struct {
	Title            string    `json:"Title"`
	DataDir          string    `json:"DataDir"`
	Proxy            string    `json:"Proxy,omitempty"`
	WebServiceUrl    string    `json:"WebServiceUrl"`
	Dbconnect        DBConnect `json:"Dbconnect"`
	OrgTreeServerUrl string    `json:"orgTreeServerUrl"`
	SyncFiles        string    `json:"syncFiles"`
	ClientID         string    `json:"clientID"`
	ClientSecret     string    `json:"clientSecret"`
}

func main() {
	configPath := flag.String("config", "config.json", "path to config json")
	notify := flag.Bool("notify", true, "send sync result notification")
	flag.Parse()

	sysini, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("load config failed: %s\n", err)
		os.Exit(1)
	}

	if err := syncBPM(sysini); err != nil {
		message := "sync BPM personnel role failed: " + err.Error()
		fmt.Println(message)
		if *notify {
			sendLineMessage(sysini.Title, message)
		}
		os.Exit(1)
	}

	message := "sync BPM personnel role completed."
	fmt.Println(message)
	if *notify {
		sendLineMessage(sysini.Title, message)
	}
}

func loadConfig(path string) (System, error) {
	var sysini System

	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return sysini, fmt.Errorf("read %s: %w", path, err)
	}

	if err := json.Unmarshal(fileBytes, &sysini); err != nil {
		return sysini, fmt.Errorf("parse %s: %w", path, err)
	}

	return sysini, nil
}

func syncBPM(sysini System) error {
	files := splitSyncFiles(sysini.SyncFiles)
	psntree, err := NewPsnTree(sysini.Dbconnect, sysini.OrgTreeServerUrl, files)
	if err != nil {
		return err
	}

	if err := psntree.StartSync(sysini.ClientID, sysini.ClientSecret); err != nil {
		return err
	}

	return psntree.GenerateReport()
}

func splitSyncFiles(syncFiles string) []string {
	parts := strings.Split(syncFiles, ",")
	files := make([]string, 0, len(parts))
	for _, part := range parts {
		file := strings.TrimSpace(part)
		if file != "" {
			files = append(files, file)
		}
	}
	return files
}

func sendLineMessage(title, message string) {
	os.Setenv("ActionScriptURL", "https://script.google.com/macros/s/AKfycbygFuH_hX2kqYl1NWKWB0CzbWLPgjwkmCSyZG7AY6kjp5YH0Plu/exec")
	x, err := SherryErrorExecuter.NewErrorExecuter()
	if err != nil {
		fmt.Printf("create notification client failed: %s\n", err)
		return
	}
	x.Error2AS(title, fmt.Errorf("%s", message))
}
