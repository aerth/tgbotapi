package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/aerth/tgbotapi"
)

func main() {
	flag.BoolVar(&tgbotapi.Debug, "v", tgbotapi.Debug, "verbose telegram bot api logs")
	flag.Parse()
	// Debug output is slog.LevelDebug
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	// create new bot instance with http.DefaultClient and official telegram bot api server
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account @%s", bot.Self.UserName)
	// tgbotapi.Mux like http.ServeMux for slash commands (some special routes exist but are not used by default)
	mux := tgbotapi.NewMux()

	// handle slash command: /id
	mux.HandleFunc("id", func(ctx context.Context, b *tgbotapi.BotAPI, u tgbotapi.Update) error {
		_, _ = b.ReplyToM1(*u.Message, fmt.Sprintf("%d", u.Message.Chat.ID))
		return nil
	})

	// u.Message is always non-nil for this special DefaultMessagePath routing path
	mux.HandleFunc(tgbotapi.DefaultMessagePath, func(ctx context.Context, b *tgbotapi.BotAPI, u tgbotapi.Update) error {
		_ = json.NewEncoder(os.Stdout).Encode(u)
		if u.Message.Text != "" && len(u.Message.Entities) != 0 {
			msg := tgbotapi.NewMessage(u.Message.Chat.ID, u.Message.Text)
			msg.Entities = u.Message.Entities // copy entities such as bold, links, expandable_blockquote, hashtag, text_link, link_preview_options, etc
			_, _ = b.Send(msg)
		} else if u.Message.Text != "" { // without entities, echo text back
			_, _ = b.ReplyToM1(*u.Message, u.Message.Text)
		}
		return nil
	})

	ctx := context.TODO()
	updateConfig := tgbotapi.UpdateConfig{Timeout: 60}

	// start and run bot
	err = mux.Run(ctx, bot, updateConfig)
	log.Fatalf("fatal: %v", err)
}
