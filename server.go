package caller

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/smalls0098/caller/apipb"
)

func serverClient(proxy string) *http.Client {
	if len(proxy) == 0 {
		return client
	}
	u, err := url.Parse(proxy)
	if err != nil {
		return client
	}
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second, // 请求超时
				KeepAlive: 10 * time.Second, // 检测连接是否存活
			}).DialContext,
			ForceAttemptHTTP2:     false,
			MaxIdleConns:          50,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Proxy: func(req *http.Request) (*url.URL, error) {
				return u, nil
			},
		},
	}
}

func Server(w http.ResponseWriter, r *http.Request, debug bool) {
	if r.Body == nil {
		writeErr(w, errors.New("missing body"))
		return
	}
	defer r.Body.Close()
	callBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, errors.New("missing body"))
		return
	}

	callReq := &apipb.CallRequest{}
	if err = unmarshal(callBody, callReq); err != nil {
		writeErr(w, err)
		return
	}
	if len(callReq.GetMethod()) == 0 {
		writeErr(w, errors.New("method is nil"))
		return
	}
	if len(callReq.GetUrl()) == 0 {
		writeErr(w, errors.New("url is nil"))
		return
	}
	if !strings.HasPrefix(callReq.GetUrl(), "http") {
		writeErr(w, errors.New("url is nil"))
		return
	}

	if debug {
		log.Printf("method: %s, url: %s", callReq.GetMethod(), callReq.GetUrl())
	}

	req, err := http.NewRequestWithContext(r.Context(), callReq.GetMethod(), callReq.GetUrl(), nil)
	if err != nil {
		writeErr(w, err)
		return
	}
	body := io.NopCloser(http.NoBody)
	if len(callReq.GetBody()) > 0 {
		body = io.NopCloser(bytes.NewReader(callReq.GetBody()))
	}
	req.Header = protoHeader2header(callReq.GetHeaders())
	req.Body = body

	// 自定义serverClient，加入代理支持
	resp, err := serverClient(callReq.GetProxy()).Do(req)
	if err != nil {
		writeErr(w, err)
		return
	}

	respBody := make([]byte, 0)
	if resp.Body != nil {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if len(b) > 0 {
			respBody = b
		}
	}

	pr, err := marshal(&apipb.CallResponse{
		StatusCode:    int32(resp.StatusCode),
		Headers:       header2protoHeader(resp.Header),
		ContentLength: resp.ContentLength,
		Body:          respBody,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(pr)
}

func writeErr(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err.Error()))
}
