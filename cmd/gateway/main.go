package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"foundryprotocol/gateway"
)

func main() {
	configPath := flag.String("config", "gateway/gateway.yaml", "path to gateway config")
	flag.Parse()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

	cfg, err := gateway.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load gateway config")
	}
	reg, err := gateway.LoadRegistry(cfg.ServersFile)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load server registry")
	}
	srv := gateway.NewServer(cfg, reg, logger)
	httpServer := &http.Server{
		Addr:    cfg.Addr,
		Handler: srv.Handler(),
	}

	logger.Info().Str("addr", cfg.Addr).Int("servers", len(reg.List())).Msg("gateway started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("gateway crashed")
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	logger.Info().Msg("gateway stopped")
}
