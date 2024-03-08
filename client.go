package caller

import (
	"bytes"
	"io"
	"net/http"

	"github.com/smalls0098/caller/apipb"
)

func Client(apiUrl string, proxy string) Caller {
	return func(c *http.Client, r *http.Request) (*http.Response, error) {
		if len(apiUrl) == 0 {
			return c.Do(r) // 系统执行
		}
		body := make([]byte, 0)
		if r.Body != nil {
			defer r.Body.Close()
			b, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			if len(b) > 0 {
				body = b
			}
		}

		callReq, err := marshal(&apipb.CallReq{
			Method:  r.Method,
			Url:     r.URL.String(),
			Headers: header2protoValue(r.Header),
			Body:    body,
			Proxy:   proxy,
		})
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, apiUrl, bytes.NewReader(callReq))
		if err != nil {
			return nil, err
		}
		return client.Do(req)
	}
}
