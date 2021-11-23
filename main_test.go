package main

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGetTitle(t *testing.T) {
	testString := `<title>TestTitle</title>
		<a href="localhost1">linkText</a>`

	reader := strings.NewReader(testString)

	page, _ := NewPage(reader)
	correctResult := "TestTitle"
	result := page.GetTitle()

	if correctResult != result {
		t.Errorf("Wrong title. Need %s got %s ", correctResult, result)
	}
}

func TestGetLinks(t *testing.T) {

	testString := `<title>TestTitle</title>
		<a href="localhost1">linkText</a>
		<a href="localhost2">linkText</a>`

	reader := strings.NewReader(testString)

	page, _ := NewPage(reader)

	correctResult := []string{"localhost1", "localhost2"}
	result := page.GetLinks()

	assert.Equal(t, correctResult, result)
}

type testRequester struct {
	timeout time.Duration
	t       *testing.T
}

func newTestRequester(t *testing.T, timeout time.Duration) *testRequester {
	return &testRequester{timeout: timeout, t: t}
}

func (r *testRequester) Get(ctx context.Context, url string) (Page, error) {
	r.t.Log("testRequester start")
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		cl := &httpClient{
			Timeout: r.timeout,
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		body, err := cl.Do(req)
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
	return nil, nil
}

type httpClient struct {
	Timeout time.Duration
}

type testResponse struct {
	Body io.ReadCloser
}

func (*httpClient) Do(req *http.Request) (*testResponse, error) {
	url := req.URL.String()

	var refs []string
	switch url {
	case "localhost":
		refs = []string{"1", "2"}
	default:
		num, _ := strconv.Atoi(url)
		refs = []string{strconv.Itoa(num * 10), strconv.Itoa(num*10 + 1)}
	}
	testString := `<title>TestTitle</title>
		<a href="` + refs[0] + `">linkText</a>
		<a href="` + refs[1] + `">linkText</a>`

	reader := ioutil.NopCloser(strings.NewReader(testString))

	return &testResponse{Body: reader}, nil
}

func TestGet(t *testing.T) {
	cfg := Config{
		MaxDepth:     1,
		DepthIncSize: 2,
		MaxResults:   10,
		MaxErrors:    5,
		MaxDuration:  10,
		Url:          "localhost",
		Timeout:      10,
		logLevel:     logrus.DebugLevel,
	}

	var r Requester

	ctx, _ := context.WithTimeout(context.Background(), time.Duration(cfg.MaxDuration)*time.Second)

	r = newTestRequester(t, time.Duration(cfg.Timeout)*time.Second)
	page, _ := r.Get(ctx, cfg.Url)

	correctResult := []string{"1", "2"}
	result := page.GetLinks()

	assert.Equal(t, correctResult, result)

}

func TestScan(t *testing.T) {
	cfg := Config{
		MaxDepth:     3,
		DepthIncSize: 2,
		MaxResults:   10,
		MaxErrors:    5,
		MaxDuration:  5,
		Url:          "localhost",
		Timeout:      10,
		logLevel:     logrus.DebugLevel,
	}
	var cr Crawler
	var log = logrus.New()
	var r Requester
	r = newTestRequester(t, time.Duration(cfg.Timeout)*time.Second)
	cr = NewCrawler(r, cfg.MaxDepth, log)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MaxDuration)*time.Second)
	select {
	case <-ctx.Done():
		t.Log("ctx DONE")
	default:
		t.Log("ctx isn't DONE")
	}
	correctResult := []string{"localhost", "1", "2", "21", "20", "10", "11"}

	t.Log("go")
	go cr.Scan(ctx, cfg.Url, 0)
	scannedUrls := testProcessResult(t, ctx, cancel, cr, cfg, log)
	t.Log("chekResult")

	sort.Strings(correctResult)
	sort.Strings(scannedUrls)
	assert.Equal(t, correctResult, scannedUrls)
}

func testProcessResult(t *testing.T, ctx context.Context, cancel func(), cr Crawler, cfg Config, log *logrus.Logger) []string {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors

	scannedUrls := []string{}
	for {
		t.Log("processResult")
		select {
		case <-ctx.Done():
			return scannedUrls
		case msg := <-cr.ChanResult():

			if msg.Err != nil {
				maxErrors--
				log.Errorf("crawler result return err: %s\n", msg.Err.Error())
				if maxErrors <= 0 {
					cancel()
					return scannedUrls
				}
			} else {
				scannedUrls = append(scannedUrls, msg.Url)
				maxResult--
				log.Infof("crawler result: [url: %s] Title: %s\n", msg.Url, msg.Title)
				if maxResult <= 0 {
					cancel()
					return scannedUrls
				}
			}
		}
	}
}
