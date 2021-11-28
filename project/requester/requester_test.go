package requester

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
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

type testResponse struct {
	Body io.ReadCloser
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

	ctx, _ := context.WithTimeout(context.Background(), time.Duration(cfg.MaxDuration)*time.Second)

	r := NewRequester(&httpClient{})
	page, _ := r.Get(ctx, cfg.Url)

	correctResult := []string{"1", "2"}
	result := page.GetLinks()

	assert.Equal(t, correctResult, result)

}
