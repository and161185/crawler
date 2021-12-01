package crawler

import (
	"context"
	"net/http"
	"sync"

	"project/requester"

	"github.com/sirupsen/logrus"
)

type Requester interface {
	Get(context.Context, string) (requester.Page, error)
}

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type CrawlResult struct {
	Err   error
	Title string
	Url   string
}

type crawler struct {
	r        Requester
	res      chan CrawlResult
	visited  map[string]struct{}
	mu       sync.RWMutex
	maxDepth int
	log      *logrus.Logger
}

func (c *crawler) MaxDepth() int {
	return c.maxDepth
}

func (c *crawler) IncMaxDepth(val int) {
	c.maxDepth = +val
}

func NewCrawler(cl HttpClient, maxDepth int, log *logrus.Logger) *crawler {
	return &crawler{
		r:        requester.NewRequester(cl),
		res:      make(chan CrawlResult),
		visited:  make(map[string]struct{}),
		mu:       sync.RWMutex{},
		maxDepth: maxDepth,
		log:      log,
	}
}

func (c *crawler) Scan(ctx context.Context, url string, depth int) {

	maxDepth := c.MaxDepth()

	debugFields := logrus.Fields{
		"url":      url,
		"depth":    depth,
		"maxDepth": maxDepth,
	}

	//panic simulation
	/*defer func() {
		if r := recover(); r != nil {
			c.log.WithFields(debugFields).Panic()
		}
	}()
	if depth > 2 {
		panic("panic simulation")
	}*/
	//-panic simulation

	if depth >= c.maxDepth { //Проверяем то, что есть запас по глубине
		c.log.WithFields(debugFields).Debug("maxDepth is reached")
		return
	}
	c.mu.RLock()
	_, ok := c.visited[url] //Проверяем, что мы ещё не смотрели эту страницу
	c.mu.RUnlock()
	if ok {
		c.log.WithFields(debugFields).Debug("url already visited")
		return
	}
	select {
	case <-ctx.Done(): //Если контекст завершен - прекращаем выполнение
		c.log.WithFields(debugFields).Debug("context is done")
		return
	default:

		page, err := c.r.Get(ctx, url) //Запрашиваем страницу через Requester
		if err != nil {
			c.res <- CrawlResult{Err: err} //Записываем ошибку в канал
			c.log.WithFields(debugFields).Debug(err)
			return
		}
		c.mu.Lock()
		c.visited[url] = struct{}{} //Помечаем страницу просмотренной
		c.mu.Unlock()
		c.res <- CrawlResult{ //Отправляем результаты в канал
			Title: page.GetTitle(),
			Url:   url,
		}
		for _, link := range page.GetLinks() {
			c.log.WithFields(debugFields).Debugf("Start scan %s", link)
			go c.Scan(ctx, link, depth+1) //На все полученные ссылки запускаем новую рутину сборки
		}

	}
}

func (c *crawler) ChanResult() <-chan CrawlResult {
	return c.res
}
