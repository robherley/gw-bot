package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/robherley/gw-bot/internal/db"
	"github.com/robherley/gw-bot/internal/db/sqlgen"
)

func NewUnsubscribe(db db.DB) Handler {
	return &Unsubscribe{db}
}

type Unsubscribe struct {
	db db.DB
}

func (cmd *Unsubscribe) Name() string {
	return "unsubscribe"
}

func (cmd *Unsubscribe) Description() string {
	return "Unsubscribe from search terms(s)."
}

func (cmd *Unsubscribe) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		userID := UserID(i)
		if userID == "" {
			return nil
		}

		subs, err := cmd.db.FindUserSubscriptions(ctx, userID)
		if err != nil {
			return err
		}

		if len(subs) == 0 {
			return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "ℹ️ You have no subscriptions to unsubscribe from.",
				},
			})
		}

		options := make([]discordgo.SelectMenuOption, 0, len(subs))
		for _, sub := range subs {
			term := sub.Term
			if term == "" {
				term = "<empty>"
			}

			options = append(options, discordgo.SelectMenuOption{
				Label: term,
				Value: sub.ID,
			})
		}

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				CustomID: cmd.Name(),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.SelectMenu{
								CustomID:    cmd.Name() + ":remove",
								Placeholder: "⏹️ What subscription(s) would you like to remove?",
								Options:     options,
								MaxValues:   len(options),
							},
						},
					},
				},
			},
		})
	case discordgo.InteractionMessageComponent:
		subIDs := i.MessageComponentData().Values
		userID := UserID(i)

		subscriptions, err := cmd.db.FindUserSubscriptions(ctx, userID)
		if err != nil {
			return err
		}

		if err := cmd.db.DeleteUserSubscriptions(ctx, sqlgen.DeleteUserSubscriptionsParams{
			UserID: userID,
			Ids:    subIDs,
		}); err != nil {
			return err
		}

		deleted := make([]sqlgen.Subscription, 0, len(subIDs))
		for _, subID := range subIDs {
			for _, sub := range subscriptions {
				if sub.ID == subID {
					deleted = append(deleted, sub)
				}
			}
		}

		builder := strings.Builder{}
		builder.WriteString("🔕 Unsubscribed from ")
		builder.WriteString(strconv.Itoa(len(deleted)))
		builder.WriteString(" terms(s):\n")

		for _, sub := range deleted {
			builder.WriteString("- \"")
			builder.WriteString(sub.ID)
			builder.WriteString("\"\n")
		}

		dm, err := s.UserChannelCreate(userID)
		if err != nil {
			return err
		}

		_, err = s.ChannelMessageSend(dm.ID, builder.String())
		if err != nil {
			return err
		}

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ Unsubscribed from %d term(s)!", len(deleted)),
			},
		})
	default:
		return nil
	}
}
