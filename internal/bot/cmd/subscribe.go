package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/mattn/go-sqlite3"
	"github.com/robherley/gw-bot/internal/db"
	"github.com/robherley/gw-bot/internal/db/sqlgen"
	"github.com/robherley/gw-bot/internal/gw"
)

func NewSubscribe(db db.DB, gw *gw.Client) Handler {
	return &Subscribe{db, gw}
}

type Subscribe struct {
	db db.DB
	gw *gw.Client
}

func (cmd *Subscribe) Name() string {
	return "subscribe"
}

func (cmd *Subscribe) Description() string {
	return "Subscribe to a search term."
}

func (cmd *Subscribe) Options() []*discordgo.ApplicationCommandOption {
	termMinLength := 1
	termMaxLength := 100
	notifyMinValue := float64(1)
	return []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "term",
			Description: "What items do you want to look for?",
			MinLength:   &termMinLength,
			MaxLength:   termMaxLength,
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "min",
			Description: "Minimum price to alert on",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "max",
			Description: "Maximum price to alert on",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "notify",
			Description: "How many minutes before the auction ends to send a notification",
			Required:    false,
			MinValue:    &notifyMinValue,
		},
		// TODO: category
	}
}

func (cmd *Subscribe) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		userID := UserID(i)
		if userID == "" {
			return nil
		}

		data := i.ApplicationCommandData()

		var (
			term          string
			minPrice      *int64
			maxPrice      *int64
			notifyMinutes int64 = 10
		)

		for _, option := range data.Options {
			switch option.Name {
			case "term":
				term = option.StringValue()
			case "min":
				min := option.IntValue()
				minPrice = &min
			case "max":
				max := option.IntValue()
				maxPrice = &max
			case "notify":
				notifyMinutes = option.IntValue()
			}
		}

		if minPrice != nil && maxPrice != nil {
			if *minPrice > *maxPrice {
				return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "⛔ Minimum price must be less than or equal to maximum price.",
					},
				})
			}
		}

		subs, err := cmd.db.FindUserSubscriptions(ctx, userID)
		if err != nil {
			return err
		}

		if len(subs) >= 25 {
			return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "⛔ You can only have up to 25 subscriptions at a time. Use `/subscriptions` to see your current subscriptions and `/unsubscribe` to remove one.",
				},
			})
		}

		sub, err := cmd.db.CreateSubscription(ctx, sqlgen.CreateSubscriptionParams{
			ID:            db.NewID(),
			UserID:        userID,
			Term:          term,
			MinPrice:      minPrice,
			MaxPrice:      maxPrice,
			NotifyMinutes: notifyMinutes,
		})

		if err != nil {
			var sqliteErr sqlite3.Error
			if errors.As(err, &sqliteErr) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
				return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("⛔ Already subscribed for search: %q.\nSee subscriptions with `/subscriptions` and `/unsubscribe` if you wish to change your configured subscriptions.", term),
					},
				})
			}
			return err
		}

		log := slog.With("subscription_id", sub.ID, "user_id", sub.UserID)
		log.Info("created subscription")

		msg := fmt.Sprintf("🔔 Subscribed for term: %q\n", term)
		if sub.MinPrice != nil || sub.MaxPrice != nil {
			msg += "\n"
			if sub.MaxPrice == nil {
				msg += fmt.Sprintf("Will only alert on items $%d or more", *sub.MinPrice)
			} else if sub.MinPrice == nil {
				msg += fmt.Sprintf("Will only alert on items $%d or less", *sub.MaxPrice)
			} else {
				msg += fmt.Sprintf("Will only alert on items $%d - $%d", *sub.MinPrice, *sub.MaxPrice)
			}
		}

		dm, err := s.UserChannelCreate(userID)
		if err != nil {
			return err
		}

		_, err = s.ChannelMessageSend(dm.ID, msg)
		if err != nil {
			return err
		}

		n, err := cmd.seedItems(ctx, sub)
		if err != nil {
			log.Error("failed to seed items", "error", err)
		}

		log.Info("seeded items", "count", n)

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ Subscribed, <@%s>! You will receive a DM when new items are found.", userID),
			},
		})
	default:
		return nil
	}
}

func (cmd *Subscribe) seedItems(ctx context.Context, sub sqlgen.Subscription) (int, error) {
	opts := gw.SearchOptionsFromSubscription(sub)

	items := map[int64]gw.Item{}

	newestItems, err := cmd.gw.Search(ctx, sub.Term, append(opts, gw.WithDescending(true))...)
	if err != nil {
		return 0, err
	}

	for _, item := range newestItems {
		items[item.ItemID] = item
	}

	endingSoonItems, err := cmd.gw.Search(ctx, sub.Term, append(opts, gw.WithDescending(false))...)
	if err != nil {
		return 0, err
	}

	for _, item := range endingSoonItems {
		items[item.ItemID] = item
	}

	for _, item := range items {
		if item.Ended() {
			continue
		}

		_, err := cmd.db.CreateItem(ctx, item.NewCreateItemParams(sub))
		if err != nil {
			return 0, err
		}
	}

	return len(items), nil
}
