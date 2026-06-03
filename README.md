# reportcli

`reportcli` 是一個以 Go 撰寫的 BPM 人員角色同步工具。程式會依照設定檔連線至組織樹資料服務，取得指定的遠端角色人員 JSON，並與 BPM MySQL 資料庫中的角色成員資料進行比對，最後同步更新 `doreuserrole`。

## 主要功能

- 透過 `clientID` / `clientSecret` 向組織樹服務取得 access token。
- 支援一次同步多個遠端 JSON 檔案，檔名以逗號分隔設定於 `syncFiles`。
- 依遠端資料的 `typez` 對應 BPM 資料表 `role.namez`，取得要同步的 `roleID`。
- 以遠端人員的 `SysId` 優先比對 `doreuser.ssoID`。
- 若找不到 `ssoID`，改用 `Name` 與 `depID` 比對 `doreuser.name`、`doreuser.depID`。
- 當既有人員的 `ssoID`、部門名稱或部門代碼需要更新時，自動更新 `doreuser`。
- 將 BPM 角色成員同步成遠端名單：
  - 遠端名單沒有、但資料庫角色中存在的人員，會從 `doreuserrole` 刪除。
  - 遠端名單存在、但資料庫角色中不存在的人員，會新增至 `doreuserrole`。
  - 找不到對應 `usrNo` 的遠端人員不會新增角色關聯。
- 同步成功或失敗時可發送通知；也可用 `-notify=false` 關閉。

## 專案結構

```text
.
├── sync_bpm.go              # CLI 入口、設定讀取、同步流程啟動、通知
├── psntree.go               # 組織樹 API 串接與 BPM 角色成員同步邏輯
├── db.go                    # MySQL 連線設定與 DSN 建立
├── client/                  # HTTP client、Bearer token 結構與請求工具
├── errorexecuter/           # 通知與錯誤訊息傳送工具
├── config.json              # 執行設定範例
├── *_test.go                # 單元測試
└── TEST_PLAN.md             # 測試計畫
```

## 系統需求

- Go 1.25 或以上版本
- MySQL
- 可連線至組織樹資料服務
- BPM 資料庫需具備下列資料表與欄位：
  - `role(roleID, namez)`
  - `doreuser(usrNo, ssoID, department, depID, name)`
  - `doreuserrole(usrNo, roleID)`

## 建置

在專案根目錄執行：

```powershell
go test ./...
go build -o reportcli.exe .
```

若需要使用專案內的 Go cache 目錄，可先設定：

```powershell
$env:GOCACHE="D:\myprograms\reportcli\.gocache"
$env:GOMODCACHE="D:\myprograms\reportcli\.gomodcache"
go test ./...
go build -o reportcli.exe .
```

## 執行方式

使用預設設定檔 `config.json`：

```powershell
.\reportcli.exe
```

指定設定檔：

```powershell
.\reportcli.exe -config .\config.json
```

關閉同步結果通知：

```powershell
.\reportcli.exe -config .\config.json -notify=false
```

### CLI 參數

| 參數 | 預設值 | 說明 |
| --- | --- | --- |
| `-config` | `config.json` | 指定 JSON 設定檔路徑。 |
| `-notify` | `true` | 是否在同步完成或失敗時發送通知。 |

## 設定檔格式

設定檔為 JSON 格式。範例：

```json
{
  "Title": "BPM personnel role sync",
  "DataDir": "",
  "Proxy": "",
  "WebServiceUrl": "",
  "Dbconnect": {
    "DBMSType": "MySQL",
    "DBMS": "mysql",
    "DbServer": "127.0.0.1",
    "DbPort": "3306",
    "DbName": "bpm",
    "DbLogin": "bpm_user",
    "DbPasswd": "password",
    "Charset": "utf8",
    "ParseTime": "True",
    "Loc": "Local"
  },
  "orgTreeServerUrl": "https://example.org/datahubcenter/",
  "clientID": "bpm",
  "clientSecret": "secret",
  "syncFiles": "CyberSecurityITs.json,CyberSecurityManager.json"
}
```

### 設定欄位

| 欄位 | 必填 | 說明 |
| --- | --- | --- |
| `Title` | 否 | 通知訊息使用的系統名稱。 |
| `DataDir` | 否 | 目前程式未直接使用，保留作為既有設定欄位。 |
| `Proxy` | 否 | 目前程式未直接使用，保留作為既有設定欄位。 |
| `WebServiceUrl` | 否 | 目前同步流程未直接使用，保留作為既有設定欄位。 |
| `Dbconnect` | 是 | BPM MySQL 連線設定。 |
| `orgTreeServerUrl` | 是 | 組織樹服務根 URL。程式會呼叫 `{orgTreeServerUrl}accesstoken` 與 `{orgTreeServerUrl}read/{fileName}`。 |
| `clientID` | 是 | 取得 access token 的 client ID。 |
| `clientSecret` | 是 | 取得 access token 的 client secret。 |
| `syncFiles` | 是 | 要同步的遠端 JSON 檔名，多個檔案以逗號分隔。空白會自動 trim。 |

### 資料庫連線欄位

| 欄位 | 必填 | 預設值 | 說明 |
| --- | --- | --- | --- |
| `DBMS` | 否 | `mysql` | 目前僅支援 `mysql`。 |
| `DBMSType` | 否 | 無 | 保留欄位，不影響連線。 |
| `DbServer` | 是 | 無 | MySQL 主機位址。 |
| `DbPort` | 是 | 無 | MySQL 連接埠。 |
| `DbName` | 是 | 無 | 資料庫名稱。 |
| `DbLogin` | 是 | 無 | 資料庫帳號。 |
| `DbPasswd` | 否 | 空字串 | 資料庫密碼。 |
| `Charset` | 否 | `utf8` | MySQL DSN charset。 |
| `ParseTime` | 否 | `True` | MySQL DSN parseTime。 |
| `Loc` | 否 | `Local` | MySQL DSN loc。 |

產生的 MySQL DSN 格式如下：

```text
{DbLogin}:{DbPasswd}@tcp({DbServer}:{DbPort})/{DbName}?charset={Charset}&parseTime={ParseTime}&loc={Loc}
```

## 遠端 API 規格

### 取得 Access Token

程式會向下列 URL 發送 GET 請求：

```text
{orgTreeServerUrl}accesstoken
```

Request body 為 JSON：

```json
{
  "clientID": "bpm",
  "clientSecret": "secret"
}
```

預期成功回應：

```json
{
  "accessToken": "token-value"
}
```

若回應包含 `errMsg`，程式會視為取 token 失敗：

```json
{
  "errMsg": "error message"
}
```

### 讀取遠端人員 JSON

每個 `syncFiles` 檔案會呼叫：

```text
{orgTreeServerUrl}read/{fileName}
```

Header：

```text
Authorization: Bearer {accessToken}
Content-Type: multipart/form-data
```

Multipart 欄位：

| 欄位 | 值 |
| --- | --- |
| `user` | `bpm` |
| `action` | `GET` |
| `file` | 目前同步的檔名 |
| `object` | `/read/{fileName}` |
| `systemName` | `opendatacenter` |

預期回應為人員陣列：

```json
[
  {
    "depID": "D001",
    "depName": "資訊處",
    "typez": "CyberSecurityITs",
    "SysId": "sso-id",
    "Name": "王小明",
    "usrNo": "",
    "op": "",
    "inDate": "2026-06-03"
  }
]
```

## 同步規則

1. 每個遠端 JSON 檔案會獨立同步一次。
2. 程式使用該檔案第一筆人員資料的 `typez` 查詢 BPM 角色：

```sql
select roleID from role where namez=?
```

3. 對遠端每一筆人員資料，先以 `SysId` 查詢 `doreuser.ssoID`：

```sql
select usrNo,ssoID,department,depID from doreuser where ssoID=?
```

4. 若查無資料，改用姓名與部門代碼查詢：

```sql
select usrNo,ssoID from doreuser where name=? and depID=?
```

5. 若查到 `usrNo`，且 `ssoID`、`department` 或 `depID` 需要補齊或更新，程式會執行：

```sql
update doreuser set ssoID=?,department=?,depID=? where usrNo=?
```

6. 程式列出資料庫目前該角色成員：

```sql
select c.usrNo,c.depID,c.department,c.ssoID,c.name
from doreuserrole b, doreuser c
where b.usrNo=c.usrNo and b.roleID=?
```

7. 資料庫中存在、遠端名單不存在的成員會刪除：

```sql
delete from doreuserrole where usrNo=? and roleID=?
```

8. 遠端名單存在、資料庫中不存在的成員會新增：

```sql
insert into doreuserrole(usrNo, roleID) values(?, ?)
```

## 錯誤處理

常見錯誤包含：

| 條件 | 錯誤訊息 |
| --- | --- |
| `orgTreeServerUrl` 空白 | `ServerUrl is empty` |
| `syncFiles` 空白或只包含空白項目 | `files is empty, please check syncFiles` |
| `clientID` 或 `clientSecret` 空白 | `client ID or client Secret is empty` |
| 資料庫設定不完整 | `database config is incomplete` |
| `DBMS` 不是 `mysql` | `DBMS {value} not supported` |
| access token 回應空白 | `get access token from {url} returned empty string` |
| access token 回應包含 `errMsg` | `get access token error: {message}` |
| 遠端檔案回應空白 | `remote file {fileName} result is empty` |
| 遠端 JSON 無法解析 | `parse remote file {fileName}: ...` |
| `typez` 空白 | `typez is empty` |
| 找不到對應角色 | `role not found for typez {typez}` |

同步失敗時程式會輸出錯誤並以 exit code `1` 結束。

## 通知

預設 `-notify=true`。同步完成會送出：

```text
sync BPM personnel role completed.
```

同步失敗會送出：

```text
sync BPM personnel role failed: {error}
```

通知流程會使用 `errorexecuter`，並在程式內設定 `ActionScriptURL`。`ErrorLineNotifyToken` 與 `ErrorLineNotifyURL` 可透過環境變數覆寫；若未設定，程式會使用內建預設值。

```powershell
$env:ErrorLineNotifyToken="your-token"
$env:ErrorLineNotifyURL="https://notify-api.line.me/api/notify"
.\reportcli.exe -config .\config.json
```

如不需要通知，請使用：

```powershell
.\reportcli.exe -config .\config.json -notify=false
```

## 測試

執行全部單元測試：

```powershell
go test ./...
```

目前測試涵蓋：

- `splitSyncFiles`：逗號分隔、空白 trim、忽略空項目。
- `loadConfig`：讀取合法設定檔與處理無效 JSON。
- `mysqlDSN`：預設與自訂 MySQL DSN 選項。
- `NewPsnTree` / `StartSync`：必要欄位驗證。
- `syncRoleMembers`：新增與刪除角色成員的 SQL 行為。

## 使用注意事項

- `config.json` 可能包含資料庫密碼、client secret 或通知 token，請勿提交真實正式環境機敏資訊。
- 程式會直接修改 BPM 資料庫的 `doreuser` 與 `doreuserrole`，正式執行前建議先備份資料或在測試環境驗證。
- `typez` 必須能對應到 `role.namez`，否則該檔案無法同步。
- 遠端人員若無法在 `doreuser` 找到對應 `usrNo`，不會被新增至 `doreuserrole`。
- HTTP client 目前略過 TLS 憑證驗證，若部署於正式環境，建議評估安全性並調整實作。

## 程式運作流程圖

```mermaid
flowchart TD
    A([開始執行 reportcli.exe]) --> B[解析 CLI 參數<br/>-config / -notify]
    B --> C[讀取 config.json]
    C --> D{設定檔是否讀取成功?}
    D -- 否 --> E[輸出 load config failed]
    E --> F{notify=true?}
    F -- 是 --> G[發送失敗通知]
    F -- 否 --> H([exit code 1])
    G --> H

    D -- 是 --> I[解析 syncFiles<br/>逗號分隔並移除空白]
    I --> J[建立 PsnTree]
    J --> K{orgTreeServerUrl 與 syncFiles 是否有效?}
    K -- 否 --> L[回傳設定錯誤]
    L --> M[輸出 sync BPM personnel role failed]
    M --> N{notify=true?}
    N -- 是 --> O[發送失敗通知]
    N -- 否 --> H
    O --> H

    K -- 是 --> P{clientID / clientSecret 是否有效?}
    P -- 否 --> L
    P -- 是 --> Q[呼叫 orgTreeServerUrl + accesstoken<br/>取得 access token]
    Q --> R{token 回應是否成功?}
    R -- 否 --> L
    R -- 是 --> S[保存 Bearer token]

    S --> T{{逐一處理 syncFiles}}
    T --> U[呼叫 orgTreeServerUrl + read/fileName<br/>讀取遠端人員 JSON]
    U --> V{遠端 JSON 是否有效且非空?}
    V -- 否 --> L
    V -- 是 --> W[解析人員陣列 people]

    W --> X{people 是否為空陣列?}
    X -- 是 --> T
    X -- 否 --> Y[連線 BPM MySQL]
    Y --> Z{資料庫連線是否成功?}
    Z -- 否 --> L

    Z -- 是 --> AA[取第一筆 people[0].typez]
    AA --> AB[查詢 role.roleID<br/>where role.namez = typez]
    AB --> AC{是否找到 roleID?}
    AC -- 否 --> AD[回傳 role not found for typez]
    AD --> M

    AC -- 是 --> AE{{逐一比對遠端人員}}
    AE --> AF[先用 SysId 查詢 doreuser.ssoID]
    AF --> AG{是否找到使用者?}
    AG -- 否 --> AH[改用 Name + depID 查詢 doreuser]
    AG -- 是 --> AI[取得 usrNo / ssoID / department / depID]
    AH --> AJ{是否找到使用者?}
    AJ -- 否 --> AK[usrNo 維持空白<br/>後續不新增角色]
    AJ -- 是 --> AI
    AI --> AL{ssoID / department / depID<br/>是否需要補齊或更新?}
    AL -- 是 --> AM[更新 doreuser<br/>ssoID / department / depID]
    AL -- 否 --> AN[寫回 people[i].usrNo]
    AM --> AN
    AK --> AO{是否還有遠端人員?}
    AN --> AO
    AO -- 是 --> AE

    AO -- 否 --> AP[查詢資料庫目前角色成員<br/>doreuserrole + doreuser]
    AP --> AQ{{同步角色成員}}
    AQ --> AR[資料庫成員存在但遠端名單不存在]
    AR --> AS[刪除 doreuserrole]
    AS --> AT[遠端名單存在但資料庫成員不存在]
    AQ --> AT
    AT --> AU{遠端人員是否有 usrNo?}
    AU -- 否 --> AV[略過該人員]
    AU -- 是 --> AW[新增 doreuserrole]
    AV --> AX{此檔案同步完成?}
    AW --> AX
    AX -- 否 --> AQ
    AX -- 是 --> AY[重新查詢同步後角色成員]
    AY --> AZ{新增/刪除結果是否正確?}
    AZ -- 否 --> L
    AZ -- 是 --> BA{是否還有下一個 syncFile?}
    BA -- 是 --> T

    BA -- 否 --> BB[輸出 sync BPM personnel role completed.]
    BB --> BC{notify=true?}
    BC -- 是 --> BD[發送完成通知]
    BC -- 否 --> BE([正常結束])
    BD --> BE
```
