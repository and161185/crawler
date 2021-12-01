//File is not `gofmt`-ed with `-s` (gofmt)
//gofmt -s -w .\requester\requester.go
package requester

import (
	"context"
	"io"
	"net/http"

	"project/page"
)

type Page interface {
	GetTitle() string
	GetLinks() []string
}

type Requester struct {
	cl HttpClient
}

func NewPage(raw io.Reader) (Page, error) {
	return page.NewPage(raw)
}

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

func NewRequester(cl HttpClient) *Requester {
	return &Requester{cl: cl}
}

func (r *Requester) Get(ctx context.Context, url string) (Page, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		body, err := r.cl.Do(req)
		if err != nil {
			return nil, err
		}
		defer body.Body.Close()
		page, err := NewPage(body.Body)
		if err != nil {
			return nil, err
		}
		return page, nil
	}
	//unreachable: unreachable code (govet)
}
