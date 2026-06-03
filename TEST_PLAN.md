# 測試方案

## 目標

確認 `reportcli` 可以正常編譯、讀取設定、產生資料庫連線字串，並正確判斷 BPM 角色成員需要新增或刪除的資料。

## 快速測試

```powershell
$env:GOCACHE="D:\myprograms\reportcli\.gocache"
$env:GOMODCACHE="D:\myprograms\reportcli\.gomodcache"
$env:GOSUMDB="off"
go test ./...
go build -o reportcli.exe .
```

預期結果：

- `go test ./...` 顯示所有 package 通過。
- `go build -o reportcli.exe .` 產生 `reportcli.exe`。

## 單元測試範圍

- `splitSyncFiles`：確認逗號分隔、空白修剪、空項目忽略。
- `loadConfig`：確認可讀取合法 JSON，且非法 JSON 會回傳錯誤。
- `mysqlDSN`：確認預設與自訂 MySQL DSN 選項。
- `NewPsnTree` / `StartSync`：確認必要欄位缺漏時會失敗。
- `syncRoleMembers`：使用假的 DB executor 驗證新增與刪除 SQL 會被正確呼叫。

## 手動整合測試

1. 準備測試用 MySQL，不建議使用正式資料庫。
2. 建立或複製以下測試資料表：`role`、`doreuser`、`doreuserrole`。
3. 建立測試角色，`role.namez` 必須對應遠端 JSON 的 `typez`。
4. 建立測試使用者，至少涵蓋：
   - 已有 `ssoID` 且資料一致的使用者。
   - 只有姓名與部門代碼可匹配的使用者。
   - 遠端名單不存在、但資料庫角色內仍存在的使用者。
5. 將 `config.json` 指向測試資料庫與測試組織樹服務。
6. 執行：

```powershell
.\reportcli.exe -config .\config.json -notify=false
```

預期結果：

- 遠端名單存在、資料庫角色不存在的人員會新增到 `doreuserrole`。
- 遠端名單不存在、資料庫角色仍存在的人員會從 `doreuserrole` 刪除。
- 無法匹配到 `usrNo` 的遠端人員不會新增。
- 終端顯示 `sync BPM personnel role completed.`。

## 失敗案例測試

- `syncFiles` 空白：應顯示 `files is empty, please check syncFiles`。
- `orgTreeServerUrl` 空白：應顯示 `ServerUrl is empty`。
- `clientID` 或 `clientSecret` 空白：應顯示 `client ID or client Secret is empty`。
- 遠端服務回傳空字串：應顯示遠端檔案或 token 回傳空值的錯誤。
- `typez` 找不到對應 `role.namez`：應顯示 role not found。

## 注意事項

- 整合測試請加上 `-notify=false`，避免測試失敗時送出通知。
- 單元測試不會連線外部 API 或資料庫。
- `.gocache` 與 `.gomodcache` 是本機建置快取，可刪除後重新產生。
