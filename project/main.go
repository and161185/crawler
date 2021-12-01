package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project/crawler"

	"github.com/sirupsen/logrus"
)

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, depth int)
	ChanResult() <-chan crawler.CrawlResult
	MaxDepth() int
	IncMaxDepth(val int)
}

//`NewCrawler` is unused (deadcode)

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

	log.Info("Crawler initialization")
	cr = crawler.NewCrawler(&http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second}, cfg.MaxDepth, log)
	log.Info("Crawler initialized")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MaxDuration)*time.Second)
	go cr.Scan(ctx, cfg.Url, 0)                 //Запускаем краулер в отдельной рутине
	go processResult(ctx, cancel, cr, cfg, log) //Обрабатываем результаты в отдельной рутине

	sigCh := make(chan os.Signal, 1)      //sigchanyzer: misuse of unbuffered os.Signal channel as argument to signal.Notify (govet)
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
