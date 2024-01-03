package caller

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func Test_caller(t *testing.T) {
	caller := Caller(func(c *http.Client, req *http.Request) (*http.Response, error) {
		return Client("http://127.0.0.1:13802/call")(c, req)
	})
	params := url.Values{}
	params.Set("username", "smalls")
	req, err := http.NewRequest(http.MethodPost, "https://httpbin.org/post", strings.NewReader(params.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := caller(client, req)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(resp.Body)
	t.Log(resp.StatusCode)
	t.Log(resp.ContentLength)
	t.Log(string(data))
}
