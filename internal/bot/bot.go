package bot

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/bwmarrin/discordgo"
	"github.com/robherley/gw-bot/internal/bot/cmd"
	"github.com/robherley/gw-bot/internal/db"
)

// MaxMessagesPerNotify is the maximum number of messages to send in a single notify.
// This number was based on the discord maximum of 10 embeds per message.
const MaxMessagesPerNotify = 10

type Bot struct {
	DB db.DB

	session  *discordgo.Session
	handlers map[string]cmd.Handler
}

func New(token string, db db.DB) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	session.UserAgent = "github.com/robherley/gw-bot (https://github.com/robherley/gw-bot)"

	b := &Bot{
		DB:      db,
		session: session,
	}

	b.handlers = buildHandlers(
		cmd.NewPing(),
		cmd.NewSubscribe(db),
		cmd.NewUnsubscribe(db),
		cmd.NewSubscriptions(db),
	)

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
		ctx := context.Background()

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
			if err := handler.Handle(ctx, s, i); err != nil {
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
			if err := handler.Handle(ctx, s, i); err != nil {
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

func (b *Bot) NotifyNewItems() error {
	return nil
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

func buildHandlers(handlers ...cmd.Handler) map[string]cmd.Handler {
	m := make(map[string]cmd.Handler, len(handlers))
	for _, h := range handlers {
		m[h.Name()] = h
	}
	return m
}
