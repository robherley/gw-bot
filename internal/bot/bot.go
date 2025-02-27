package bot

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robherley/gw-bot/internal/bot/cmd"
	"github.com/robherley/gw-bot/internal/db"
	"github.com/robherley/gw-bot/internal/db/sqlgen"
	"github.com/robherley/gw-bot/internal/gw"
)

// MaxMessagesPerNotify is the maximum number of messages to send in a single notify.
// This number was based on the discord maximum of 10 embeds per message.
const MaxMessagesPerNotify = 10

type Bot struct {
	db       db.DB
	gw       *gw.Client
	ctx      context.Context
	session  *discordgo.Session
	handlers map[string]cmd.Handler
}

func New(ctx context.Context, token string, db db.DB, gw *gw.Client) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	session.UserAgent = "github.com/robherley/gw-bot (https://github.com/robherley/gw-bot)"

	b := &Bot{
		db:      db,
		gw:      gw,
		ctx:     ctx,
		session: session,
	}

	b.handlers = make(map[string]cmd.Handler)
	for _, handler := range []cmd.Handler{
		cmd.NewPing(),
		cmd.NewSubscribe(db, gw),
		cmd.NewUnsubscribe(db),
		cmd.NewSubscriptions(db),
	} {
		b.handlers[handler.Name()] = handler
	}

	return b, nil
}

func (b *Bot) Start() (err error) {
	if err := b.session.Open(); err != nil {
		return err
	}

	b.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		slog.Info("ready to go", "bot_user", r.User.String())
	})

	b.session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		log := LogWith(i, "interaction_type", i.Type.String())

		defer func() {
			if r := recover(); r != nil {
				log.Error("panic", "err", r, "stack", string(debug.Stack()))
			}
		}()

		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			handler, ok := b.handlers[i.ApplicationCommandData().Name]
			if !ok {
				log.Warn("no handler found")
				return
			}

			log.Info("invoking command")
			if err := handler.Handle(b.ctx, s, i); err != nil {
				log.Error("failed", "err", err)
			}
		case discordgo.InteractionMessageComponent:
			customID := i.MessageComponentData().CustomID
			log = log.With("custom_id", customID)

			cmd, _ := cmd.FromCustomID(customID)
			handler, ok := b.handlers[cmd]
			if !ok {
				log.Warn("no handler found")
				return
			}

			log.Info("invoking command")
			if err := handler.Handle(b.ctx, s, i); err != nil {
				log.Error("failed", "err", err)
			}
		default:
			log.Warn("unknown interaction type")
		}
	})

	return nil
}

func (b *Bot) Close() error {
	return b.session.Close()
}

func (b *Bot) Unregister(guild string) error {
	if guild == "" {
		return nil
	}

	if guild == "global" {
		guild = ""
	}

	appID := b.session.State.User.ID
	existing, err := b.session.ApplicationCommands(appID, guild)
	if err != nil {
		return err
	}

	for _, cmd := range existing {
		log := slog.With("cmd", cmd.Name, "guild_id", guild)
		if err := b.session.ApplicationCommandDelete(appID, guild, cmd.ID); err != nil {
			log.Error("failed to unregister")
			return err
		}
		log.Info("unregistered")
	}

	return nil
}

func (b *Bot) Register(guild string) error {
	if guild == "" {
		return nil
	}

	if guild == "global" {
		guild = ""
	}

	appID := b.session.State.User.ID
	for _, h := range b.handlers {
		log := slog.With("cmd", h.Name(), "guild_id", guild)
		_, err := b.session.ApplicationCommandCreate(
			appID,
			guild,
			cmd.ToApplicationCommand(h),
		)
		if err != nil {
			log.Error("failed to register")
			return err
		}
		log.Info("registered")
	}

	return nil
}

func (b *Bot) NotifyNewItems(sub sqlgen.Subscription, items []gw.Item) error {
	dm, err := b.session.UserChannelCreate(sub.UserID)
	if err != nil {
		return err
	}

	chunks := Chunk(items, MaxMessagesPerNotify)
	for _, chunk := range chunks {
		embeds := make([]*discordgo.MessageEmbed, 0, len(chunk))
		for _, item := range chunk {
			embeds = append(embeds, b.ItemToEmbed(item))
		}

		_, err := b.session.ChannelMessageSendComplex(dm.ID, &discordgo.MessageSend{
			Content: fmt.Sprintf("🔔 New items for %q!", sub.Term),
			Embeds:  embeds,
		})
		if err != nil {
			return err
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}

func (b *Bot) NotifyEndingSoonItems(sub sqlgen.Subscription, items []sqlgen.Item) error {
	log := slog.With("subscription_id", sub.ID, "user_id", sub.UserID)

	gwItems := make([]*gw.Item, 0, len(items))
	for _, item := range items {
		gwItem, err := b.gw.FindItem(b.ctx, item.GoodwillID)
		if err != nil {
			log.Error("failed to find item", "error", err, "goodwill_id", item.GoodwillID)
			continue
		}

		gwItems = append(gwItems, gwItem)
	}

	dm, err := b.session.UserChannelCreate(sub.UserID)
	if err != nil {
		return err
	}

	chunks := Chunk(gwItems, MaxMessagesPerNotify)
	for _, chunk := range chunks {
		embeds := make([]*discordgo.MessageEmbed, 0, len(chunk))
		for _, item := range chunk {
			embeds = append(embeds, b.ItemToEmbed(*item))
		}

		_, err := b.session.ChannelMessageSendComplex(dm.ID, &discordgo.MessageSend{
			Content: fmt.Sprintf("⏰ Items ending soon for %q!", sub.Term),
			Embeds:  embeds,
		})
		if err != nil {
			return err
		}

		time.Sleep(2 * time.Second)
	}
	return nil
}

func (b *Bot) ItemToEmbed(item gw.Item) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: item.Title,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Current Price",
				Value:  fmt.Sprintf("$%.2f", item.CurrentPrice),
				Inline: true,
			},
			{
				Name:   "Ends",
				Value:  item.RelativeEndTime(),
				Inline: true,
			},
			{
				Name:   "Bids",
				Value:  strconv.FormatInt(item.NumBids, 10),
				Inline: true,
			},
			{
				Name:   "Category",
				Value:  item.CategoryName,
				Inline: true,
			},
			{
				Name:   "Kind",
				Value:  item.Kind(),
				Inline: true,
			},
		},
		URL: item.URL(),
	}

	if item.ImageURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: item.ImageURL,
		}
	}

	slog.Info("embedding item",
		"item_id", item.ItemID,
		"url", item.URL(),
		"image_url", item.ImageURL,
	)

	return embed
}

func Chunk[T any](slice []T, chunkSize int) [][]T {
	chunks := make([][]T, 0, (len(slice)+chunkSize-1)/chunkSize)

	for chunkSize < len(slice) {
		slice, chunks = slice[chunkSize:], append(chunks, slice[0:chunkSize:chunkSize])
	}
	chunks = append(chunks, slice)

	return chunks
}
