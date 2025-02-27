package cmd

import (
	"context"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/robherley/gw-bot/internal/db"
)

func NewSubscriptions(db db.DB) Handler {
	return &Subscriptions{db}
}

type Subscriptions struct {
	db db.DB
}

func (cmd *Subscriptions) Name() string {
	return "subscriptions"
}

func (cmd *Subscriptions) Description() string {
	return "View active subscriptions."
}

func (cmd *Subscriptions) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if i.Type != discordgo.InteractionApplicationCommand {
		return nil
	}

	userID := UserID(i)
	if userID == "" {
		return nil
	}

	subs, err := cmd.db.FindUserSubscriptions(ctx, userID)
	if err != nil {
		return err
	}

	builder := strings.Builder{}
	builder.WriteString("You have ")
	builder.WriteString(strconv.Itoa(len(subs)))
	builder.WriteString(" subscription(s)")

	if len(subs) > 0 {
		builder.WriteString(":\n")
		for _, sub := range subs {
			builder.WriteString("- ")
			builder.WriteString(sub.Term)

			if sub.MinPrice != nil || sub.MaxPrice != nil {
				builder.WriteString("$")
				if sub.MinPrice != nil {
					builder.WriteString(strconv.FormatInt(*sub.MinPrice, 10))
				} else {
					builder.WriteString("0")
				}
				builder.WriteString(" - $")
				if sub.MaxPrice != nil {
					builder.WriteString(strconv.FormatInt(*sub.MaxPrice, 10))
				} else {
					builder.WriteString("∞")
				}
				builder.WriteString(" ")
			}

			builder.WriteString("\n")
		}
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			CustomID: cmd.Name(),
			Content:  builder.String(),
		},
	})
}
