# Golang bindings for the Telegram Bot API

[![Go Reference](https://pkg.go.dev/badge/github.com/aerth/tgbotapi.svg)](https://pkg.go.dev/github.com/aerth/tgbotapi)

Fork of `github.com/go-telegram-bot-api/telegram-bot-api/v5`

## New Features

- Mux routing using HandlerFunc (like http.ServeMux)
- Thumbnails
- Guest Message
- Draft Message
- Business Message
- Emoji Reactions

See [echo example](./cmd/echobot/) for Mux example

## HandlerFunc

```go
func foo(ctx context.Context, b *BotAPI, u Update) error { return nil }
mux.HandleFunc("foo", foo)
```
