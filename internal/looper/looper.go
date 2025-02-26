package looper

import (
	"context"
	"time"

	"github.com/robherley/gw-bot/internal/bot"
	"github.com/robherley/gw-bot/internal/db"
)

const (
	TickNotify  = 2 * time.Minute
	TickCleanup = 1 * time.Hour

	WindowNotify  = 5 * time.Minute
	WindowCleanup = 72 * time.Hour
)

type Looper struct {
	db  db.DB
	bot *bot.Bot
}

func New(db db.DB, bot *bot.Bot) *Looper {
	return &Looper{db, bot}
}

func (l *Looper) Notify(ctx context.Context) {

}

func (l *Looper) Cleanup(ctx context.Context) {

}
