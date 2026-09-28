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

func ParseFlags() (*Config, error) {
	urlsRaw := flag.String("urls", "", "Список URL через запятую")
	depth := flag.Int("depth", 1, "Максимальная глубина рекурсивного обхода")
	timeout := flag.Duration("timeout", 2*time.Minute, "Общий таймаут выполнения")
	reqTimeout := flag.Duration("request-timeout", 10*time.Second, "Таймаут одного HTTP-запроса")
	output := flag.String("output", "result.json", "Путь к JSON-файлу с результатами")
	logFile := flag.String("log", "crawler.log", "Путь к файлу логов")

	flag.Parse()

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
