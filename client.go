package caller

import (
	"bytes"
	"errors"
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

		callReq, err := marshal(&apipb.CallRequest{
			Method:  r.Method,
			Url:     r.URL.String(),
			Headers: header2protoHeader(r.Header),
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
		req.Header.Set("Content-Type", "application/gpb")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.Body == nil {
			return resp, err
		}
		defer resp.Body.Close()
		callBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		callResp := &apipb.CallResponse{}
		if err = unmarshal(callBody, callResp); err != nil {
			return nil, err
		}
		if callResp.GetStatusCode() == 0 {
			return nil, errors.New("status code is 0")
		}

		resp.StatusCode = int(callResp.GetStatusCode())
		resp.Header = protoHeader2header(callResp.GetHeaders())
		resp.ContentLength = callResp.GetContentLength()
		if len(callResp.Body) > 0 {
			resp.Body = io.NopCloser(bytes.NewReader(callResp.GetBody()))
		}

		return resp, err
	}
}
