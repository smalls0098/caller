package caller

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/smalls0098/caller/apipb"
)

func buildServerHttpClient(proxy string) *http.Client {
	if len(proxy) == 0 {
		return client
	}
	u, err := url.Parse(proxy)
	if err != nil {
		return client
	}
	t := transport.Clone()
	t.Proxy = func(req *http.Request) (*url.URL, error) {
		return u, nil
	}
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: t,
	}
}

func Server(rw http.ResponseWriter, r *http.Request) {
	ServerWithInterceptor(rw, r, nil)
}

func ServerWithInterceptor(rw http.ResponseWriter, r *http.Request, interceptor Interceptor) {
	ctx := r.Context()

	params, err := parseParams(r)
	if err != nil {
		writeErr(rw, err)
		return
	}

	// 请求
	req, err := makeReq(ctx, params)
	if err != nil {
		writeErr(rw, err)
		return
	}
	if interceptor != nil {
		req, err = interceptor.OnBefore(req)
		if err != nil {
			writeErr(rw, err)
			return
		}
	}

	// 结果
	res, err := buildServerHttpClient(params.GetProxy()).Do(req)
	if err != nil {
		writeErr(rw, err)
		return
	}
	if interceptor != nil {
		res, err = interceptor.OnAfter(res)
		if err != nil {
			writeErr(rw, err)
			return
		}
	}

	removeHopByHopHeaders(res.Header)
	copyHeader(rw.Header(), res.Header)
	rw.WriteHeader(res.StatusCode)

	if res.Body != nil {
		defer func() {
			_, _ = io.CopyN(io.Discard, res.Body, 1024*4)
			_ = res.Body.Close()
		}()
		_, err = io.Copy(rw, res.Body)
		if err != nil {
			writeErr(rw, err)
			return
		}
	}
}

func parseParams(r *http.Request) (*apipb.CallReq, error) {
	if r.Body == nil || r.ContentLength <= 0 {
		return nil, errors.New("missing body")
	}

	defer r.Body.Close()
	params := &apipb.CallReq{}
	if body, err := io.ReadAll(r.Body); err != nil {
		return nil, err
	} else {
		if err = unmarshal(body, params); err != nil {
			return nil, err
		}
	}

	if len(params.GetMethod()) == 0 {
		return nil, errors.New("method is nil")
	}
	if len(params.GetUrl()) <= 4 || !strings.HasPrefix(strings.ToLower(params.GetUrl()[:4]), "http") {
		return nil, errors.New("url is nil")
	}
	return params, nil
}

func makeReq(ctx context.Context, params *apipb.CallReq) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, params.GetMethod(), params.GetUrl(), nil)
	if err != nil {
		return nil, err
	}

	// set body
	if len(params.GetBody()) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(params.GetBody()))
	} else {
		req.Body = http.NoBody
	}

	// set header
	req.Header = protoValue2header(params.GetHeaders())
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "")
	}
	removeHopByHopHeaders(req.Header)
	return req, nil
}

func writeErr(w http.ResponseWriter, err error) {
	w.Header().Set("is_err", "1")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err.Error()))
	log.Printf("writeErr: %+v", err)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func removeHopByHopHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = textproto.TrimString(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
	for _, f := range hopHeaders {
		h.Del(f)
	}
}
