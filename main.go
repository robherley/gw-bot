package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/lmittmann/tint"
	"github.com/robherley/gw-bot/internal/bot"
	"github.com/robherley/gw-bot/internal/db"
)

//go:embed database/migrations/*.sql
var migrations embed.FS

type Config struct {
	DiscordToken string `desc:"API Token for Discord" required:"true"`
	DatabaseFile string `desc:"Path of SQLite database file" default:"gw-bot.db" required:"false"`
}

func init() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))
}

func main() {
	if err := run(); err != nil {
		slog.Error("gwb failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{}

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags]\n\n", os.Args[0])
		envconfig.Usagef("", &cfg, flag.CommandLine.Output(), envconfig.DefaultListFormat)
		fmt.Fprintln(flag.CommandLine.Output(), "\nFlags:")
		flag.PrintDefaults()
	}

	register := flag.String("register", "", "guild to register commands (or 'global')")
	unregister := flag.String("unregister", "", "guild to unregister commands (or 'global')")
	flag.Parse()

	if err := envconfig.Process("", &cfg); err != nil {
		return err
	}

	db, err := db.NewSQLite(cfg.DatabaseFile)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Migrate(ctx, migrations); err != nil {
		return err
	}

	bot, err := bot.New(cfg.DiscordToken, db)
	if err != nil {
		return err
	}
	defer bot.Close()

	if err := bot.Start(); err != nil {
		return err
	}

	exitEarly := false

	if *register != "" {
		if err := bot.Register(*register); err != nil {
			return err
		}
		exitEarly = true
	}

	if *unregister != "" {
		if err := bot.Unregister(*unregister); err != nil {
			return err
		}
		exitEarly = true
	}

	if exitEarly {
		return nil
	}

	slog.Info("github.com/robherley/gw-bot is initialized")

	// l := looper.New(db, bot)
	// go l.Notify(ctx)
	// go l.Cleanup(ctx)

	wait()
	return nil
}

func wait() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	sig := <-done
	slog.Warn("received signal, shutting down", "signal", sig.String())
}
