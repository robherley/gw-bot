package looper

import (
	"context"
	"log/slog"
	"time"

	"github.com/robherley/gw-bot/internal/bot"
	"github.com/robherley/gw-bot/internal/db"
	"github.com/robherley/gw-bot/internal/db/sqlgen"
	"github.com/robherley/gw-bot/internal/gw"
)

const (
	TickNotifyNew    = 5 * time.Minute
	TickNotifyEnding = 1 * time.Minute
	TickCleanup      = 1 * time.Hour
)

type Looper struct {
	db  db.DB
	bot *bot.Bot
	gw  *gw.Client
}

func New(db db.DB, bot *bot.Bot, gw *gw.Client) *Looper {
	return &Looper{db, bot, gw}
}

func (l *Looper) NotifyNewItems(ctx context.Context) {
	ticker := time.NewTicker(TickNotifyNew)
	defer ticker.Stop()

	log := slog.With("component", "looper.notify.new")
	log.Info("starting loop", "tick", TickNotifyNew)

	for {
		select {
		case <-ticker.C:
			subscriptions, err := l.db.FindSubscriptionsToNotify(ctx)
			if err != nil {
				log.Error("failed to find subscriptions to notify", "error", err)
				continue
			}

			for _, sub := range subscriptions {
				time.Sleep(2 * time.Second)

				opts := gw.SearchOptionsFromSubscription(sub)
				opts = append(opts, gw.WithDescending(true))

				log := log.With("subscription_id", sub.ID, "user_id", sub.UserID)

				items, err := l.gw.Search(ctx, sub.Term, opts...)
				if err != nil {
					log.Error("failed to search for items", "error", err)
					continue
				}

				if len(items) == 0 {
					log.Info("no new items found")
					continue
				}

				log.Info("new items found", "count", len(items))

				for _, item := range items {
					_, err := l.db.CreateItem(ctx, item.NewCreateItemParams(sub))
					if err != nil {
						log.Error("failed to create item", "error", err)
						continue
					}
				}

				if err := l.bot.NotifyNewItems(sub, items); err != nil {
					log.Error("failed to notify new items", "error", err)
				}

				if err := l.db.SetSubscriptionLastNotifiedAt(ctx, sub.ID); err != nil {
					log.Error("failed to set last notified at", "error", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (l *Looper) NotifyEndingSoonItems(ctx context.Context) {
	ticker := time.NewTicker(TickNotifyEnding)
	defer ticker.Stop()

	log := slog.With("component", "looper.notify.ending")
	log.Info("starting loop", "tick", TickNotifyEnding)

	for {
		select {
		case <-ticker.C:
			items, err := l.db.FindItemsEndingSoon(ctx)
			if err != nil {
				log.Error("failed to find items ending soon", "error", err)
				continue
			}

			sub2items := map[string][]sqlgen.Item{}
			for _, item := range items {
				sub2items[item.SubscriptionID] = append(sub2items[item.SubscriptionID], item)
			}

			for subID, items := range sub2items {
				time.Sleep(2 * time.Second)

				sub, err := l.db.FindSubscription(ctx, subID)
				if err != nil {
					log.Error("failed to find subscription", "error", err)
					continue
				}

				if err := l.bot.NotifyEndingSoonItems(sub, items); err != nil {
					log.Error("failed to notify ending soon items", "error", err)
				}

				itemIDs := make([]string, 0, len(items))
				for _, item := range items {
					itemIDs = append(itemIDs, item.ID)
				}

				if err := l.db.SetItemSentFinal(ctx, itemIDs); err != nil {
					log.Error("failed to set item sent final notification", "error", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (l *Looper) Cleanup(ctx context.Context) {
	ticker := time.NewTicker(TickCleanup)
	defer ticker.Stop()

	log := slog.With("component", "looper.cleanup")
	log.Info("starting loop", "tick", TickCleanup)

	for {
		select {
		case <-ticker.C:
			n, err := l.db.DeleteExpiredItems(ctx)
			if err != nil {
				log.Error("failed to delete expired items", "error", err)
			}
			log.Info("deleted expired items", "count", n)
		case <-ctx.Done():
			return
		}
	}
}
