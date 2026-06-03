/*
   func: 檔案下載
   參考資訊：https://blog.devgenius.io/concurrent-file-download-with-go-495d7b946492
             https://github.com/cheikhshift/medium_examples/blob/main/concurrent-download/main.go
*/

package SherryClient

import (
/*
   "fmt"
   "io"
   "io/ioutil"
   // "log"
   "net/http"
   "strconv"
*/
)

// 檔案分割的資訊內容
type Part struct {
   Data  []byte
   Index int
}

/*
// 下載工作
func(app *SryClient) download(index, size int, c chan Part, url string) {
   client := &http.Client{}
   start := index * size
   dataRange := fmt.Sprintf("bytes=%d-%d", start, start+size-1)

   if index == app.Workers-1 {
      dataRange = fmt.Sprintf("bytes=%d-", start)
   }
   req, err := http.NewRequest("GET", url, nil)
   if err != nil {
      return  // code to restart download
   }
   req.Header.Add("Range", dataRange)
   resp, err := client.Do(req)
   if err != nil { // code to restart download
      return
   }
   defer resp.Body.Close()
   body, err := io.ReadAll(resp.Body)
   if err != nil { // code to restart download
      return
   }
   c <- Part{Index: index, Data: body}
}

func(app *SryClient) Download(url string)(error) {
   var size int
   results := make(chan Part, app.Workers)
   parts := [app.Workers][]byte{}

   req, err := http.NewRequest("HEAD", url, nil)
   if err != nil {
      return err
   }
   resp, err := app.Request.Client.Do(req)
   if err != nil {
      return err
   }
   if header, ok := resp.Header["Content-Length"]; ok {
      fileSize, err := strconv.Atoi(header[0])

      if err != nil {
         return fmt.Errorf("File size could not be determined : %s", err.Error())
      }
      size = fileSize / workers
   } else {
      return fmt.Errorf("File size was not provided!")
   }
   for i := 0; i < app.Workers; i++ {
      go download(i, size, results, url)
   }
   counter := 0
   for part := range results {
      counter++
      parts[part.Index] = part.Data
      if counter == app.Workers {
         break
      }
   }
   file := []byte{}
   for _, part := range parts {
      file = append(file, part...)
   }
   if err := ioutil.WriteFile("./data.zip", file, 0700); err != nil {
      return err
   }

   return nil
}
*/
