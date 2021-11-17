package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

type CrawlResult struct {
	Err   error
	Title string
	Url   string
}

type Page interface {
	GetTitle() string
	GetLinks() []string
}

type page struct {
	doc *goquery.Document
}

func NewPage(raw io.Reader) (Page, error) {
	doc, err := goquery.NewDocumentFromReader(raw)
	if err != nil {
		return nil, err
	}
	return &page{doc: doc}, nil
}

func (p *page) GetTitle() string {
	return p.doc.Find("title").First().Text()
}

func (p *page) GetLinks() []string {
	var urls []string
	p.doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		url, ok := s.Attr("href")
		if ok {
			urls = append(urls, url)
		}
	})
	return urls
}

type Requester interface {
	Get(ctx context.Context, url string) (Page, error)
}

type requester struct {
	timeout   time.Duration
	userDepth int
}

func NewRequester(timeout time.Duration) *requester {
	return &requester{timeout: timeout}
}

func (r *requester) Get(ctx context.Context, url string) (Page, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		cl := &http.Client{
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

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, depth int)
	ChanResult() <-chan CrawlResult
	MaxDepth() int
	IncMaxDepth(val int)
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

func NewCrawler(r Requester, maxDepth int, log *logrus.Logger) *crawler {
	return &crawler{
		r:        r,
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
	defer func() {
		if r := recover(); r != nil {
			c.log.WithFields(debugFields).Panic()
		}
	}()
	if depth > 0 {
		panic("panic simulation")
	}
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

//Config - структура для конфигурации
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

func main() {

	cfg := Config{
		MaxDepth:     3,
		DepthIncSize: 2,
		MaxResults:   10,
		MaxErrors:    5,
		MaxDuration:  10,
		Url:          "https://telegram.org",
		Timeout:      10,
		logLevel:     logrus.DebugLevel,
	}

	var log = logrus.New()
	log.Formatter = new(logrus.JSONFormatter)
	log.Level = cfg.logLevel

	var cr Crawler
	var r Requester

	log.Info("Requester initialization")
	r = NewRequester(time.Duration(cfg.Timeout) * time.Second)
	log.Info("Requester initialized")

	log.Info("Crawler initialization")
	cr = NewCrawler(r, cfg.MaxDepth, log)
	log.Info("Crawler initialized")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MaxDuration)*time.Second)
	go cr.Scan(ctx, cfg.Url, 0)                 //Запускаем краулер в отдельной рутине
	go processResult(ctx, cancel, cr, cfg, log) //Обрабатываем результаты в отдельной рутине

	sigCh := make(chan os.Signal)         //Создаем канал для приема сигналов
	signal.Notify(sigCh, syscall.SIGINT)  //Подписываемся на сигнал SIGINT
	signal.Notify(sigCh, syscall.SIGUSR1) //Подписываемся на сигнал SIGUSR1

	for {
		select {
		case <-ctx.Done(): //Если всё завершили - выходим
			fmt.Println(ctx.Err())
			return
		case syssig := <-sigCh:
			switch syssig {
			case syscall.SIGINT:
				log.Info("got SIGINT")
				cancel() //Если пришёл сигнал SigInt - завершаем контекст
			case syscall.SIGUSR1:
				log.Info("got SIGUSR1")
				cr.IncMaxDepth(cfg.DepthIncSize)
			}

		}
	}
}

func processResult(ctx context.Context, cancel func(), cr Crawler, cfg Config, log *logrus.Logger) {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cr.ChanResult():
			if msg.Err != nil {
				maxErrors--
				log.Errorf("crawler result return err: %s\n", msg.Err.Error())
				if maxErrors <= 0 {
					cancel()
					return
				}
			} else {
				maxResult--
				log.Infof("crawler result: [url: %s] Title: %s\n", msg.Url, msg.Title)
				if maxResult <= 0 {
					cancel()
					return
				}
			}
		}
	}
}
