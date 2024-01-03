package caller

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smalls0098/caller/apipb"
)

type Caller func(c *http.Client, req *http.Request) (*http.Response, error)

var client = &http.Client{
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
	},
}

func header2protoHeader(h http.Header) map[string]*apipb.HeaderValue {
	res := map[string]*apipb.HeaderValue{}
	for k, vs := range h {
		res[k] = &apipb.HeaderValue{Items: vs}
	}
	return res
}

func protoHeader2header(h map[string]*apipb.HeaderValue) http.Header {
	res := http.Header{}
	for k, vs := range h {
		for _, s := range vs.Items {
			res.Add(k, s)
		}
	}
	return res
}

func gzipEnc(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Flush(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func gzipDec(reader io.Reader) ([]byte, error) {
	gr, err := gzip.NewReader(reader)
	if err != nil {
		var out []byte
		return out, err
	}
	defer gr.Close()
	return io.ReadAll(gr)
}

func unmarshal(b []byte, m proto.Message) error {
	data, err := gzipDec(bytes.NewReader(b))
	if err != nil {
		return err
	}
	return proto.Unmarshal(data, m)
}

func marshal(m proto.Message) ([]byte, error) {
	data, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}
	return gzipEnc(data)
}
