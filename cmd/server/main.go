package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/wyw14/cry-055/internal/application"
	"github.com/wyw14/cry-055/internal/config"
	"github.com/wyw14/cry-055/internal/platform/clock"
	"github.com/wyw14/cry-055/internal/platform/localfile"
	"github.com/wyw14/cry-055/internal/platform/localnotify"
	"github.com/wyw14/cry-055/internal/repository/postgres"
	httptransport "github.com/wyw14/cry-055/internal/transport/http"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load configuration", zap.Error(err))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("open database", zap.Error(err))
	}
	defer store.Close()
	files, err := localfile.New(cfg.AttachmentRoot, cfg.MaxUploadBytes)
	if err != nil {
		logger.Fatal("open attachment store", zap.Error(err))
	}
	systemClock := clock.System{}
	notifier := localnotify.New()
	catalog := application.NewCatalogService(store, store, systemClock)
	instruments := application.NewInstrumentService(store, store, store, systemClock)
	plans := application.NewPlanService(store, store, systemClock)
	executions := application.NewExecutionService(store, store, store, store, systemClock)
	nonconformance := application.NewNonconformanceService(store, store, store, store, systemClock)
	usage := application.NewUsageService(store, store, systemClock, 30)
	alerts := application.NewAlertService(store, store, store, notifier, systemClock)
	reports := application.NewReportService(store, store, store, store)
	_ = application.NewCertificateService(store, files, systemClock)
	router := httptransport.NewRouter(httptransport.Services{Catalog: catalog, Instruments: instruments, Plans: plans, Executions: executions, Nonconformance: nonconformance, Usage: usage, Alerts: alerts, Reports: reports}, httptransport.Options{Logger: logger, RequestTimeout: cfg.RequestTimeout, AllowedOrigin: "http://localhost:5173", Ready: func() error {
		readyCtx, readyCancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		defer readyCancel()
		return store.Ping(readyCtx)
	}})
	server := &http.Server{Addr: cfg.Address, Handler: router, ReadHeaderTimeout: cfg.RequestTimeout, ReadTimeout: cfg.RequestTimeout, WriteTimeout: cfg.RequestTimeout * 2, IdleTimeout: cfg.RequestTimeout * 12}
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		logger.Info("server listening", zap.String("address", cfg.Address))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()
	<-stopped
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
}
