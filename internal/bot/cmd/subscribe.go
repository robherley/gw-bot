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
)

func NewSubscribe(db db.DB) Handler {
	return &Subscribe{db, nil}
}

type Subscribe struct {
	db   db.DB
	opts []discordgo.SelectMenuOption
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
			term     string
			minPrice *int64
			maxPrice *int64
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

		sub, err := cmd.db.CreateSubscription(ctx, sqlgen.CreateSubscriptionParams{
			ID:       db.NewID(),
			UserID:   userID,
			Term:     term,
			MinPrice: minPrice,
			MaxPrice: maxPrice,
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

		slog.Info("created subscription", "subscription_id", sub.ID, "user_id", sub.UserID)

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
