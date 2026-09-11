package tgbotapi

import (
	"context"
	"errors"
	"log/slog"
)

type Mux struct {
	routes      map[string]Handler
	BaseContext context.Context
	NewContext  func(ctx context.Context, u Update) context.Context
}

func NewMux() *Mux {
	return &Mux{routes: make(map[string]Handler)}
}

func (m *Mux) HandleFunc(path string, h HandlerFunc) {
	m.Handle(path, h)
}

func (m *Mux) Handle(path string, h Handler) {
	if _, ok := m.routes[path]; ok {
		panic("already")
	}
	m.routes[path] = h
}

type Handler interface {
	HandleUpdate(ctx context.Context, b *BotAPI, u Update) error
}

func (mx *Mux) Run(basectx context.Context, b *BotAPI, config UpdateConfig) error {
	baser := mx.NewContext
	if baser == nil {
		baser = func(ctx context.Context, u Update) context.Context { return ctx }
	}
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

			// get route
			h, ok := mx.Route(b, u)
			if !ok {
				if b.Debug {
					slog.Debug("skipping route not found", "u", u)
				}
				continue
			}
			ctx := baser(basectx, u)

			// launchder fg or bg
			launcher := HandleLaunch(ctx, b, u, h)
			if launcher == nil { // bg
				continue
			}
			if err := launcher(); err != nil {
				slog.Error("HandleUpdate", "err", err)
				if errors.Is(err, ErrBreak) {
					return err
				}
			}
		}
	}
	return nil
}

var HandleLaunch = HandleLaunchBG

var HandleLaunchFG = func(ctx context.Context, b *BotAPI, u Update, h Handler) func() error {
	return func() error {
		return h.HandleUpdate(ctx, b, u)
	}
}

var HandleLaunchBG = func(ctx context.Context, b *BotAPI, u Update, h Handler) func() error {
	go func() {
		if err := h.HandleUpdate(ctx, b, u); err != nil {
			slog.Error("HandleUpdate", "err", err)
		}
	}()
	return nil
}

var (
	ErrNoHandlerRoute error = nil
	ErrBreak                = errors.New("break")
)

func (m Mux) Route(b *BotAPI, u Update) (Handler, bool) {
	path := ""
	if u.Message == nil {
		path = "*"
	} else if u.Message.IsCommand() {
		path = u.Message.Command()
	}
	x, ok := m.routes[path]
	if !ok && path != "" {
		x, ok = m.routes[""] // default route
	}
	return x, ok
}

var ErrOK = errors.New("ok")

type HandlerFunc func(ctx context.Context, bot *BotAPI, update Update) error

type HandlerMiddleware func(h Handler) Handler

func (h HandlerFunc) HandleUpdate(ctx context.Context, b *BotAPI, u Update) error {
	err := h(ctx, b, u)
	if err != nil {
		return err
	}
	return nil
}

func NoopHandler(ctx context.Context, b *BotAPI, u Update) error { return nil }

var _ Handler = HandlerFunc(NoopHandler)
