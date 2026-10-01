package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"crawler-cli/config"
	"crawler-cli/crawler"
	"crawler-cli/storage"
)

func main() {
	cfg, err := config.ParseFlags(os.Args[1:])
	if err != nil {
		fmt.Printf("Ошибка параметров запуска: %v\n", err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Не удалось создать лог-файл: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	fileLogger := log.New(logFile, "", log.LstdFlags)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	fmt.Println("Запуск краулера...")
	c := crawler.NewCrawler(cfg, fileLogger)

	results, err := c.Run(ctx)
	if err != nil {
		fileLogger.Printf("[CRITICAL ERROR] %v", err)
		fmt.Printf("Ошибка выполнения: %v\n", err)
		os.Exit(1)
	}

	if err := storage.SaveJSON(cfg.OutputFile, results); err != nil {
		fileLogger.Printf("[ERROR] Failed to save JSON: %v", err)
		fmt.Printf("Ошибка сохранения JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Обход завершён. Результат сохранен в %s, логи в %s\n", cfg.OutputFile, cfg.LogFile)
}
