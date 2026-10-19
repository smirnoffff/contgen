package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"contgen/internal/config"
	"contgen/internal/payment"
	"contgen/internal/storage"
	"contgen/internal/telegram"
	"contgen/internal/web"
	"contgen/internal/worker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("запуск Mini App", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cfg := config.Load()
	if cfg.DevLogin && !web.IsLoopback(cfg.ServerHost) {
		return errors.New("DEV_LOGIN requires a loopback SERVER_HOST")
	}
	db, err := storage.New(ctx, cfg.DSN())
	if err != nil {
		return err
	}
	defer db.Close()
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	err = db.Pool.Ping(pingCtx)
	pingCancel()
	if err != nil {
		return fmt.Errorf("PostgreSQL недоступен: %w", err)
	}
	adminIDs := parseIDs(cfg.AdminTelegramIDs)
	if !cfg.DevLogin && len(adminIDs) > 0 {
		if promoted, err := db.PromoteTelegramAdmins(ctx, adminIDs); err != nil {
			return fmt.Errorf("не удалось настроить администраторов: %w", err)
		} else if promoted > 0 {
			slog.Info("администраторы Telegram обновлены", "count", promoted)
		}
	}
	var bot *telegram.Client
	if cfg.TelegramBotToken != "" {
		bot, err = telegram.NewClient(cfg.TelegramBotToken)
		if err != nil {
			return err
		}
	}
	handler, err := web.New(db, web.Options{
		BotToken: cfg.TelegramBotToken, SessionSecret: cfg.SessionSecret,
		PublicURL: cfg.AppURL, DevLogin: cfg.DevLogin, DataDir: cfg.DataDir,
		Telegram: bot, AdminTelegramIDs: adminIDs, AdsEnabled: cfg.AdsEnabled, AdsGramBlockID: cfg.AdsGramBlockID, AdsGramDebug: cfg.AdsGramDebug,
	})
	if err != nil {
		return err
	}
	if bot != nil && !cfg.DevLogin {
		paymentCtx, stopPayments := context.WithCancel(ctx)
		paymentDone := make(chan struct{})
		go func() {
			defer close(paymentDone)
			if err := payment.Run(paymentCtx, bot, db); err != nil {
				slog.Error("обработчик Telegram платежей остановлен", "err", err)
			}
		}()
		defer func() { stopPayments(); <-paymentDone }()
	}
	if cfg.WorkerEnabled {
		processor, err := worker.New(db, worker.Options{DataDir: cfg.DataDir, Telegram: bot})
		if err != nil {
			return err
		}
		workerCtx, stopWorker := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			defer close(done)
			if err := processor.Run(workerCtx); err != nil {
				slog.Error("очередь видео остановлена", "err", err)
			}
		}()
		defer func() { stopWorker(); <-done }()
	}
	server := &http.Server{Addr: net.JoinHostPort(cfg.ServerHost, cfg.ServerPort), Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 150 * time.Second, IdleTimeout: time.Minute}
	failures := make(chan error, 1)
	go func() { failures <- server.ListenAndServe() }()
	slog.Info("Mini App запущен", "addr", server.Addr, "url", cfg.AppURL, "dev_login", cfg.DevLogin)
	select {
	case err := <-failures:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := server.Shutdown(shutdownCtx); err != nil {
			server.Close()
			return err
		}
		return nil
	}
}

func parseIDs(raw string) []int64 {
	var ids []int64
	for _, part := range strings.Split(raw, ",") {
		v, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil && v > 0 {
			ids = append(ids, v)
		}
	}
	return ids
}
