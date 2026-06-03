package SherryClient

import(
   // "io"
   // "fmt"
   "net/http"
   // "enclding/json"
)

type Interceptor struct {
   core http.RoundTripper
}

/*
func (Interceptor) modifyRequest(r *http.Request) *http.Request {
   reqBody := json.MustHumanize(r.Body)
   modReqBody := []byte(fmt.Sprintf(`{"req": %s}`, reqBody))
   ModReqBodyLen := len(modReqBody)
   req := r.Clone(context.Background())
   req.Body = io.NopCloser(bytes.NewReader(modReqBody))
   req.ContentLength = int64(ModReqBodyLen)
   req.Header.Set("Content-Length", fmt.Sprintf("%d", ModReqBodyLen))
   return req
}

func (i Interceptor) RoundTrip(r *http.Request) (*http.Response, error) {
   defer func() {
      _ = r.Body.Close()
   }()
   newReq := i.modifyRequest(r)     // modify before the request is sent
   return i.core.RoundTrip(newReq)  // send the request using the DefaultTransport
}
*/
