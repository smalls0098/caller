package caller

import "net/http"

type Interceptor interface {
	OnBefore(req *http.Request) (*http.Request, error)
	OnAfter(resp *http.Response) (*http.Response, error)
}
