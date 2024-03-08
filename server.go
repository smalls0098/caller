package caller

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"github.com/smalls0098/caller/apipb"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"
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

func Server(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if ctx.Done() != nil {
	} else if cn, ok := rw.(http.CloseNotifier); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		notifyChan := cn.CloseNotify()
		go func() {
			select {
			case <-notifyChan:
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	params, err := parseParams(r)
	if err != nil {
		writeErr(rw, err)
		return
	}
	req, err := makeReq(ctx, params)
	if err != nil {
		writeErr(rw, err)
		return
	}

	res, err := serverClient(params.GetProxy()).Do(req)
	if err != nil {
		writeErr(rw, err)
		return
	}

	removeHopByHopHeaders(res.Header)
	copyHeader(rw.Header(), res.Header)
	rw.WriteHeader(res.StatusCode)

	defer res.Body.Close()

	var buf []byte
	_, err = copyBuffer(rw, res.Body, buf)
	if err != nil {
		writeErr(rw, err)
		return
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
	var body io.ReadCloser = nil
	if len(params.GetBody()) > 0 {
		body = io.NopCloser(bytes.NewReader(params.GetBody()))
	}
	req.Body = body
	if req.Body != nil {
		defer req.Body.Close()
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

func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if len(buf) == 0 {
		buf = make([]byte, 32*1024)
	}
	var written int64
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF && !errors.Is(rerr, context.Canceled) {

		}
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				rerr = nil
			}
			return written, rerr
		}
	}
}

func writeErr(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err.Error()))
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
