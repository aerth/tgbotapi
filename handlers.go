package tgbotapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
)

type Bot = BotAPI

type Mux struct {
	Auth        func(b *BotAPI, u Update) bool                      // First after decoding update, may return false to discard update
	BaseContext context.Context                                     // may be a mainctx
	NewContext  func(ctx context.Context, u Update) context.Context // may be used to modify a context based on the u Update.
	routes      map[string]Handler                                  //
}

var Errorf = fmt.Errorf

var (
	ErrNoHandlerRoute error = nil

	// ErrBreak may be returned by a HandlerFunc to close bot connection
	//
	// For example, a "quit'"
	ErrBreak = errors.New("break")
)

const (
	DefaultRoutePath   = ""  // non-message (some other update)
	DefaultMessagePath = "!" // message non-command "hi" but also important Message.XX members -- check msg.Text == ""
	DefaultCommandPath = "/" // message command unknown /nonexistant404
)

// CommandPaths returns all the known commands
func CommandPaths(mux *Mux) []string {
	a := slices.Collect(maps.Keys(mux.routes))
	return slices.DeleteFunc(a, func(v string) bool {
		return v == DefaultRoutePath || v == DefaultMessagePath || v == DefaultCommandPath
	})
}

func CommandPathString(v string) string {
	if v == "" {
		return "[DEFAULT-ROUTE]"
	}
	if v == "!" {
		return "[DEFAULT-MESSAGE-ROUTE]"
	}
	if v == "/" {
		return "[DEFAULT-COMMAND-ROUTE]"
	}
	// is command
	return v
}

// NewMux returns an empty mux that ignores all commands and updates
//
// Use [[Mux.Handle]] and [[Mux.HandleFunc]] to add routes.
//
// See [DefaultRoutePath] [DefaultMessagePath] and [DefaultRouteCommandPath] for special paths.
func NewMux() *Mux {
	return &Mux{routes: make(map[string]Handler), BaseContext: context.Background(), NewContext: nil, Auth: nil}
}

// HandleFunc like [http.ServeMux.HandleFunc]
func (m *Mux) HandleFunc(path string, h func(ctx context.Context, b *BotAPI, u Update) error) {
	m.Handle(path, HandlerFunc(h))
}

// HandleFunc like [http.ServeMux.HandleFunc] but for slash commands
//
// DO NOT include slash prefix in path.
//
// See [DefaultRoutePath] [DefaultMessagePath] and [DefaultRouteCommandPath] for special paths.
func (m *Mux) Handle(path string, h Handler) {
	if _, ok := m.routes[path]; ok {
		panic("command path already registered")
	}
	if len(path) > 1 && path[0] == '/' { // allow empty, allow single slash special consts
		panic("command path has slash")
	}
	m.routes[path] = h
}

// Handler similar to ServeHTTP method, we use HandleUpdate
//
// see [[HandlerFunc.HandleUpdate]]
type Handler interface {
	HandleUpdate(ctx context.Context, b *BotAPI, u Update) error
}

func (mx *Mux) Run(basectx context.Context, b *BotAPI, config UpdateConfig) error {
	baser := mx.NewContext
	if baser == nil {
		baser = func(ctx context.Context, u Update) context.Context { return ctx }
	}
	auth := mx.Auth // auth func may not change during this loop
	ch := b.GetUpdatesChan(config)
	defer b.StopReceivingUpdates()
	for !b.isShutdown.Load() {
		select {
		case <-basectx.Done():
			return context.Cause(basectx)
		case u, ok := <-ch:
			if !ok {
				return context.Cause(basectx)
			}
			if auth != nil && !auth(b, u) {
				continue
			}

			// get route
			h, ok := mx.Route(b, u)
			if !ok {
				if b.Debug {
					Slog().Debug("skipping route not found", "u", u)
				}
				continue
			}
			ctx := baser(basectx, u)

			// launcher fg or bg
			launcher := HandleLaunch(ctx, b, u, h)
			if launcher == nil { // bg
				continue
			}
			if err := launcher(); err != nil {
				Slog().Error("HandleUpdate", "err", err)
				if errors.Is(err, ErrBreak) {
					return err
				}
			}
		}
	}
	return nil
}

var Slog = slog.Default

var HandleLaunch = HandleLaunchBG

var HandleLaunchFG = func(ctx context.Context, b *BotAPI, u Update, h Handler) func() error {
	return func() error {
		return h.HandleUpdate(ctx, b, u)
	}
}

var HandleLaunchBG = func(ctx context.Context, b *BotAPI, u Update, h Handler) func() error {
	go func() {
		if err := h.HandleUpdate(ctx, b, u); err != nil {
			Slog().Error("HandleUpdate", "err", err)
		}
	}()
	return nil
}

func (m Mux) Route(b *BotAPI, u Update) (Handler, bool) {
	var path string
	switch {
	case u.Message == nil:
		path = DefaultRoutePath
	case u.Message.IsCommand():
		path = u.Message.Command() // eg: "start" or "help"
	default:
		path = DefaultMessagePath
	}
	if b.Debug {
		Slog().Debug("looking for route", "path", path)
	}
	x, ok := m.routes[path]
	if !ok && u.Message != nil && u.Message.IsCommand() {
		x, ok = m.routes[DefaultCommandPath] // default command route
		if b.Debug {
			Slog().Debug("looking for default command route", "path", path, "ok", ok)
		}
	} else {
		if b.Debug {
			Slog().Debug("looking for route", "path", path, "ok", ok)
		}
	}
	return x, ok
}

// HandlerFunc wrapper like [http.HandlerFunc]
//
// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as Update handlers. If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// [Handler] that calls f.
type HandlerFunc func(ctx context.Context, b *BotAPI, u Update) error

type HandlerMiddleware = func(h Handler) Handler

// HandleUpdate implements [[Handler]]
func (h HandlerFunc) HandleUpdate(ctx context.Context, b *BotAPI, u Update) error {
	err := h(ctx, b, u)
	if err != nil {
		return err
	}
	return nil
}

func (bot *BotAPI) ReplyToM1(replyTo Message, textV1 string) (Message, error) {
	msg := NewMessage(replyTo.Chat.ID, textV1)
	msg.ParseMode = ModeMarkdown
	msg.ReplyToMessageID = replyTo.MessageID
	sent, err := bot.Send(msg)
	if err != nil {
		return sent, err
	}
	return sent, nil
}

func (bot *BotAPI) ReplyToM2(replyTo Message, textV2 string) (Message, error) {
	msg := NewMessage(replyTo.Chat.ID, textV2)
	msg.ParseMode = ModeMarkdownV2
	msg.ReplyToMessageID = replyTo.MessageID
	sent, err := bot.Send(msg)
	if err != nil {
		return sent, err
	}
	return sent, nil
}

func (bot *BotAPI) DeleteMessage(msg Message) error {
	return nil
}

func noopHandler(ctx context.Context, b *BotAPI, u Update) error { return nil }

var _ Handler = HandlerFunc(noopHandler)
