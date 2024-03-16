package caller

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smalls0098/caller/apipb"
)

type Caller func(c *http.Client, req *http.Request) (*http.Response, error)

var transport = &http.Transport{
	DialContext: defaultTransportDialContext(&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}),
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Transport: transport,
}

func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}

func header2protoValue(h http.Header) map[string]*apipb.CallReq_HeaderValue {
	res := map[string]*apipb.CallReq_HeaderValue{}
	for k, vs := range h {
		res[k] = &apipb.CallReq_HeaderValue{Items: vs}
	}
	return res
}

func protoValue2header(h map[string]*apipb.CallReq_HeaderValue) http.Header {
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
