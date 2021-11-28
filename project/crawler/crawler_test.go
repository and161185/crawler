package crawler

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

type Config struct {
	MaxDepth     int
	DepthIncSize int
	MaxResults   int
	MaxErrors    int
	MaxDuration  int
	Url          string
	Timeout      int //in seconds
	logLevel     logrus.Level
}

type httpClient struct {
}

func (*httpClient) Do(req *http.Request) (*http.Response, error) {
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
	return &http.Response{Body: reader}, nil
}

type testResponse struct {
	Body io.ReadCloser
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

	var log = logrus.New()

	cr := NewCrawler(&httpClient{}, cfg.MaxDepth, log)

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

func testProcessResult(t *testing.T, ctx context.Context, cancel func(), cr *crawler, cfg Config, log *logrus.Logger) []string {
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
