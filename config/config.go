package config

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

type Config struct {
	URLs           []string
	Depth          int
	Timeout        time.Duration
	RequestTimeout time.Duration
	OutputFile     string
	LogFile        string
}

func ParseFlags(args []string) (*Config, error) {
	fs := flag.NewFlagSet("crawler-cli", flag.ContinueOnError)

	urlsRaw := fs.String("urls", "", "Список URL через запятую")
	depth := fs.Int("depth", 1, "Максимальная глубина рекурсивного обхода")
	timeout := fs.Duration("timeout", 2*time.Minute, "Общий таймаут выполнения")
	reqTimeout := fs.Duration("request-timeout", 10*time.Second, "Таймаут одного HTTP-запроса")
	output := fs.String("output", "result.json", "Путь к JSON-файлу с результатами")
	logFile := fs.String("log", "crawler.log", "Путь к файлу логов")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *urlsRaw == "" {
		return nil, fmt.Errorf("параметр --urls не может быть пустым")
	}

	urls := strings.Split(*urlsRaw, ",")
	for i, u := range urls {
		urls[i] = strings.TrimSpace(u)
	}

	return &Config{
		URLs:           urls,
		Depth:          *depth,
		Timeout:        *timeout,
		RequestTimeout: *reqTimeout,
		OutputFile:     *output,
		LogFile:        *logFile,
	}, nil
}
