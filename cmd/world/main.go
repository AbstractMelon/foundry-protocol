package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"foundryprotocol/server"
)

func main() {
	addr := flag.String("addr", ":8090", "listen address")
	worldName := flag.String("world", "world", "name of the world to host")
	contentDir := flag.String("content", "content", "directory with yaml content definitions")
	assetsDir := flag.String("assets", "assets", "directory with texture assets referenced by content yaml")
	saveDir := flag.String("save-dir", "saves", "directory for world save files")
	tps := flag.Int("tps", 10, "simulation ticks per second")
	dev := flag.Bool("dev", false, "run an ephemeral seeded world for development (no saves, admin commands, generous resources)")
	autoSaveTicks := flag.Int64("autosave-ticks", 600, "save every N ticks (non-dev)")
	worldSeed := flag.Int64("seed", 0, "world generation seed (0 = derive from world name)")
	flag.Parse()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

	cfg := server.Config{
		Addr:               *addr,
		WorldName:          *worldName,
		ContentDir:         *contentDir,
		AssetDir:           *assetsDir,
		SaveDir:            *saveDir,
		TPS:                *tps,
		Dev:                *dev,
		AutoSaveEveryTicks: *autoSaveTicks,
		WorldSeed:          *worldSeed,
	}

	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to start world server")
	}
	if err := srv.Listen(); err != nil {
		logger.Fatal().Err(err).Msg("failed to listen")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.Serve(ctx); err != nil {
		logger.Fatal().Err(err).Msg("world server crashed")
	}
	logger.Info().Msg("world server stopped")
}
