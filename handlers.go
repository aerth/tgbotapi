package tgbotapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"runtime/debug"
	"slices"
)

var Slog = slog.Default

type ctxk string

const CtxPath ctxk = "handler:path"

func (bot *BotAPI) GetRoutePath(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(CtxPath).(string)
	return path, ok
}

// GetMessage returns the u.Message or an empty non-nil Message
func (bot *BotAPI) GetMessage(u Update) *Message {
	if u.Message == nil {
		return &Message{}
	}
	return u.Message
}

func (bot *BotAPI) GetReply(u Update) ReplyParameters {
	if u.Message == nil || u.Message.Chat == nil {
		return ReplyParameters{}
	}
	return ReplyParameters{
		MessageID:                new(u.Message.MessageID),
		ChatID:                   u.Message.Chat.ID,
		AllowSendingWithoutReply: true,
	}
}

const (
	DefaultRoutePath   = ""  // Route Path: non-message (some other update)
	DefaultMessagePath = "!" // Route Path: message non-command "hi" but also important Message.XX members -- check msg.Text == ""
	DefaultCommandPath = "/" // Route Path: message command unknown /nonexistant404
)

type Bot = BotAPI

type Mux struct {
	Auth         func(b *BotAPI, u Update) bool                      // First after decoding update, may return false to discard update
	BaseContext  context.Context                                     // may be a mainctx
	NewContext   func(ctx context.Context, u Update) context.Context // may be used to modify a context based on the u Update.
	routes       map[string]Handler                                  //
	GetRoutePath func(u Update) (string, bool)                       // for custom non-message mini-router, see [GetRoutePath] for default if this is nil
}

var Errorf = fmt.Errorf

var (
	ErrNoHandlerRoute error = nil

	// ErrBreak may be returned by a HandlerFunc to close bot connection
	//
	// For example, a "quit" command from admin user
	ErrBreak = errors.New("break")
)

// ErrorLogger called if a Handler returns an error.
//
// This should not typically send additional telegram requests/messages.
var ErrorLogger = func(ctx context.Context, b *BotAPI, u Update, h Handler, err error) {
	path, ok := b.GetRoutePath(ctx)
	if !ok {
		path = "[unknown]"
	}
	Slog().Error("handler", "path", path, "err", fmt.Sprintf("%+v", err))
}

// HandleLaunchBG may be overrided to add a waitgroup or other sync mechanisms
var HandleLaunchBG = func(ctx context.Context, b *BotAPI, u Update, h Handler) func() error {
	go func() {
		if err := h.HandleUpdate(ctx, b, u); err != nil {
			ErrorLogger(ctx, b, u, h, err)
			if errors.Is(err, ErrBreak) {
				b.StopReceivingUpdates() // shutdown channel
			}
		}
	}()
	return nil
}

var HandleLaunchFG = func(ctx context.Context, b *BotAPI, u Update, h Handler) func() error {
	return func() error {
		err := h.HandleUpdate(ctx, b, u)
		if errors.Is(err, ErrBreak) {
			b.StopReceivingUpdates() // shutdown channel
		}
		return err
	}
}

var HandleLaunch = HandleLaunchFG

// CommandPaths returns all the known commands
func CommandPaths(mux *Mux) []string {
	a := slices.Collect(maps.Keys(mux.routes))
	return slices.DeleteFunc(a, func(v string) bool {
		return v == DefaultRoutePath || v == DefaultMessagePath || v == DefaultCommandPath
	})
}

func RoutePathString(v string) string {
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
	return &Mux{routes: make(map[string]Handler), BaseContext: context.Background(), NewContext: nil, Auth: nil, GetRoutePath: GetRoutePath}
}

// HandleFunc like [http.ServeMux.HandleFunc]
//
// DO NOT include slash prefix in path.
// See [DefaultRoutePath] [DefaultMessagePath] and [DefaultRouteCommandPath] for special paths.
func (m *Mux) HandleFunc(path string, h func(ctx context.Context, b *BotAPI, u Update) error) {
	m.Handle(path, HandlerFunc(h))
}

// Handle like [http.ServeMux.Handle] but for slash commands
//
// DO NOT include slash prefix in path.
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

var Log404 = true

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
			path, h, ok := mx.Route(b, u)
			if !ok {
				if b.Debug && Log404 {
					Slog().Debug("route not found", "u", u)
				}
				continue
			}
			ctx := baser(basectx, u)
			ctx = context.WithValue(ctx, CtxPath, path)
			if fatalerr := mx.handle(ctx, b, u, h, path); fatalerr != nil {
				return fatalerr
			}
		}
	}
	return nil
}

// handle returns fatal error only, but logs non-fatal errors
func (mx *Mux) handle(ctx context.Context, b *BotAPI, u Update, h Handler, path string) error {
	// launcher fg or bg
	launcher := HandleLaunch(ctx, b, u, h)
	if launcher == nil { // bg
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			slog.Error("recovered from panic in TG mux.Handler",
				slog.Any("panic_error", r),
				slog.String("panic_stack", string(stack)),
			)
		}
	}()
	if err := launcher(); err != nil {
		ErrorLogger(ctx, b, u, h, err)
		if errors.Is(err, ErrBreak) {
			return err
		}
	}
	return nil
}

// GetRoutePath overrides the router, may return a different route path string based on the Update.
// Essentially a mini-router, useful for non-message (such as GuestMessage, Callbacks, etc)
//
// If returned bool is false, this is ignored and routing proceeds.
// If returned bool is true, other routing logic is bypassed.
// If true and path returned is unregistered, the update will be ignored.
//
// For example: case u.GuestMessage != nil: return "_guest", true
var GetRoutePath = func(u Update) (string, bool) {
	return DefaultRoutePath, false
}

// Route an update to a Handler, using GetRoutePath first, finally falling back to DefaultRoutePath
func (m Mux) Route(b *BotAPI, u Update) (path string, h Handler, found bool) {
	if m.GetRoutePath != nil {
		custom, ok := m.GetRoutePath(u)
		if ok {
			x, ok := m.routes[custom]
			return custom, x, ok
		}
	}
	switch {
	case u.Message == nil:
		path = DefaultRoutePath
	case u.Message.IsCommand():
		path = u.Message.Command() // eg: "start" or "help"
	default:
		path = DefaultMessagePath // has a u.Message that is not a command
	}
	x, ok := m.routes[path]
	if ok || path == DefaultRoutePath { // found
		return path, x, ok
	}
	if u.Message != nil && u.Message.IsCommand() {
		x, ok = m.routes[DefaultCommandPath] // default command route
		if ok {
			return DefaultCommandPath, x, ok
		}
	}
	x, ok = m.routes[DefaultRoutePath]
	return path, x, ok
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
	if msg.Chat == nil || msg.Chat.ID == 0 {
		return Errorf("missing Chat.ID")
	}
	if msg.MessageID == 0 {
		return Errorf("missing MessageID")
	}
	if _, err := bot.Request(NewDeleteMessage(msg.Chat.ID, msg.MessageID)); err != nil {
		return Errorf("%w", err)
	}
	return nil
}

func noopHandler(ctx context.Context, b *BotAPI, u Update) error { return nil }

var _ Handler = HandlerFunc(noopHandler)
