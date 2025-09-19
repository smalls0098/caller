package caller

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/smalls0098/caller/apipb"
)

func Client(apiUrl string, proxy string) Caller {
	return ClientWithInterceptor(apiUrl, proxy, nil)
}

func ClientWithInterceptor(apiUrl string, proxy string, interceptor Interceptor) Caller {
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

		// 请求
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, apiUrl, bytes.NewReader(callReq))
		if err != nil {
			return nil, err
		}
		if interceptor != nil {
			req, err = interceptor.OnBefore(req)
			if err != nil {
				return nil, err
			}
		}

		// 结果
		res, err := DefaultHttpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if interceptor != nil {
			res, err = interceptor.OnAfter(res)
			if err != nil {
				return nil, err
			}
		}

		if res.StatusCode == http.StatusBadRequest && res.Header.Get("is_err") == "1" {
			msg := ""
			if res.Body != nil {
				data, e := io.ReadAll(res.Body)
				_ = res.Body.Close()
				if e != nil {
					return nil, e
				}
				msg = string(data)
			}
			return nil, errors.New(msg)
		}

		return res, err
	}
}
