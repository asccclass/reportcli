/*
   參考資訊：https://faun.pub/golang-http-client-interceptors-e861ec501bdf
             https://blog.devgenius.io/building-a-go-rest-client-in-2022-1ba8bb4c2201
             github.com/go-co-op/gocron
*/

package SherryClient

import (
   "io"
   "fmt"
   "time"
   "bytes"
   "strings"
   "context"
   "net/url"
   "net/http"
   "io/ioutil"
   "crypto/tls"
   "encoding/json"
   "compress/gzip"
)

type BearerToken struct {
   Token	string		`json:"accessToken"`
}

type SryClient struct {
   Request	*http.Request
   Workers	int
}

func MustHumanize(r io.Reader)(string)  {
   var m map[string]interface{}
   _ = json.NewDecoder(r).Decode(&m)
   b, _ := json.MarshalIndent(m, "", "  ")
   return string(b)
}

// PureDo
func(app *SryClient) PureGet(url string)(*http.Response, error) {
   res, err := http.Get(url)
   if err != nil {
      return nil, err
   }
   if res.StatusCode != 200 {
      return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
   }
   return res, nil
}

// 透過POST-Form送出資料
func PostForm(urlstr string, params map[string]string)(string, error) {
   // 設定參數
   requestBody := url.Values{}
   if len(params) > 0 {
      for key, val := range params {
         requestBody.Set(key, val) // requestBody.Set("client_id", h.slackClientID)
      }
   }
   resp, err := http.Post(urlstr, "application/x-www-form-urlencoded", strings.NewReader(requestBody.Encode()))
   if err != nil {
      return "", err
   }
   defer resp.Body.Close()
   body, err := ioutil.ReadAll(resp.Body)
   if err != nil {
      return "", err
   }
/*
   // set cookies
   expiration := time.Now().Add(1 * time.Hour)
   cookieS:= http.Cookie{Name: "slack_access_token", Value: slackAuthResponse.AuthedUser.AccessToken, Expires: expiration}
   http.SetCookie(w, &cookieS)
   http.Redirect(w, r, spotifyAuthURL, http.StatusSeeOther)
*/
   return string(body), nil
}

// 取得資訊
func Request(method, url, data string)(string) {
   mth := http.MethodPost

   if method == "GET" {
      mth = http.MethodGet
   } else if method == "PUT" {
   }
   req, err := http.NewRequest(mth, url, strings.NewReader(data))
   if err != nil {
      return err.Error()
   }
   transCfg := &http.Transport{
      TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
   }
   c := &http.Client {
      Transport: transCfg,
   }
   resp, err := c.Do(req)
   if err != nil {
      return err.Error()
   }
   defer func() {
       _ = resp.Body.Close()
   }()

   return MustHumanize(resp.Body)  // json 解碼
}


// 透過GET 取得相關資訊
func (app *SryClient) GetUrl(url string)(string) {
   tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
   }
   client := &http.Client{Transport: tr}
   response, err := client.Get(url)
   if err != nil {
      return err.Error()
   }
   responseData, err := ioutil.ReadAll(response.Body)
   if err != nil {
      return err.Error()
   }
   return string(responseData)
}

// 判斷是否有採用gzip壓縮
func(app *SryClient) IsGzip(res *http.Response)(bool) {
   gzipFlag := false

   for k, v := range res.Header {
      if strings.ToLower(k) == "content-encoding" && strings.ToLower(v[0]) == "gzip" {
         gzipFlag = true
         break
      }
   }
   return gzipFlag 
}

// clientDo()取得相關資訊
func(app *SryClient) clientDo(ctx context.Context, resChan chan<- string)(error) {
   app.Request = app.Request.WithContext(ctx)
   transCfg := &http.Transport{
      TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
   }
   c := &http.Client {
      Transport: transCfg,
   }
   resp, err := c.Do(app.Request)
   if err != nil {
      return err
   }
   var data []byte
   if app.IsGzip(resp) {
      gr, err := gzip.NewReader(resp.Body)
      defer gr.Close()
      if err != nil {
         return err
      }
      data, err = ioutil.ReadAll(gr)
      if err != nil {
         return err
      }
   } else {
      data, err = ioutil.ReadAll(resp.Body)
      if err != nil {
         return  err
      }
   }
   resChan <- string(data)
   return nil
}

// 執行
func(app *SryClient) Do()(string, error) {
   deadline := 15 
   d := time.Now().Add(time.Duration(deadline) * time.Second)
   resChan := make(chan string)
   ctx, cancel := context.WithDeadline(context.Background(), d)
   defer cancel()

   go app.clientDo(ctx, resChan)
   
   resData := ""
   select {
      case <-ctx.Done():
      case <-time.Tick(time.Duration(time.Duration(deadline*2) * time.Second)):
      case resData = <-resChan:
   }
   return resData, nil
   // return MustHumanize(resp.Body), nil  // json 解碼
}

// 設定 Header 內容
func(app *SryClient) AddHeader(title, value string) {
   if app.Request == nil {
      return
   }
   app.Request.Header.Set(title, value)
}

func(app *SryClient) SetRequest(req *http.Request) {
   app.Request = req
}

// Initial 初始化
func NewClient(method, url, data string)(*SryClient, error) {
   if method == "" || url == "" {
      return nil, fmt.Errorf("method or url is empty")
   }
   method = strings.ToLower(method)
   mth := ""
   if method == "get" {
      mth = http.MethodGet
   } else if method == "post" {
      mth = http.MethodPost
   } else if method == "put" {
      mth = http.MethodPut
   } else if method == "delete" {
      mth = http.MethodDelete
   } else {
      return nil, fmt.Errorf("method %s not allow", method)
   }
   // Disabling security checks is dangerous and should be avoided
   http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
   // req, err := http.NewRequest(mth, url, strings.NewReader(data))
   req, err := http.NewRequest(mth, url, bytes.NewBuffer([]byte(data)))
   if err != nil {
      return nil, err
   }
   return &SryClient {
      Request: req,
      Workers: 5,
   }, nil
}
