package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/ktr0731/go-mcp/protocol"
	"golang.org/x/exp/jsonrpc2"
)

var minimumLogLevel = new(slog.LevelVar)

// Verify that Handler implements jsonrpc2.Handler interface
var _ jsonrpc2.Handler = (*Handler)(nil)

// Handler is the main handler for MCP server implementation.
// Note that exported fields are exported for accessing by generated code. Do not access/modify them directly.
type Handler struct {
	Capabilities   protocol.ServerCapabilities
	Implementation protocol.Implementation

	Prompts       []protocol.Prompt
	PromptHandler serverHandler[protocol.GetPromptRequestParams]

	Tools       []protocol.Tool
	ToolHandler serverHandler[protocol.CallToolRequestParams]

	ResourceHandler     ServerResourceHandler
	ResourceTemplates   []ResourceTemplate
	subscribedResources sync.Map

	CompletionHandler ServerCompletionHandler

	// cancelFuncByRequestID is a map of cancellation functions for in-flight requests.
	cancelFuncByRequestID sync.Map
}

// serverHandler is a common interface for various handlers.
type serverHandler[Req any] interface {
	Handle(ctx context.Context, method string, req Req) (any, error)
}

// Handle handles an incoming request.
func (h *Handler) Handle(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	cctx, cancel := context.WithCancel(ctx)
	id := fmt.Sprintf("%v", req.ID.Raw())
	h.cancelFuncByRequestID.Store(id, cancel)
	defer h.cancelFuncByRequestID.Delete(id)

	logger := Logger(cctx, "go-mcp")

	switch {
	case req.Method == protocol.MethodPing:
		return struct{}{}, nil
	// Lifecycle: https://spec.modelcontextprotocol.io/specification/2025-03-26/basic/lifecycle/
	case req.Method == protocol.MethodInitialize:
		var params protocol.InitializeRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, jsonrpc2.ErrInvalidParams
		}
		protocolVersion := params.ProtocolVersion
		if _, ok := protocol.AvailableProtocolVersions[protocolVersion]; !ok {
			protocolVersion = protocol.LatestProtocolVersion
		}

		return &protocol.InitializeResult{
			ProtocolVersion: protocolVersion,
			Capabilities:    h.Capabilities,
			ServerInfo:      h.Implementation,
		}, nil
	case req.Method == protocol.MethodNotificationsInitialized:
		return nil, nil
	case req.Method == protocol.MethodPromptsList:
		return &listPromptsResult{Prompts: h.Prompts}, nil
	case req.Method == protocol.MethodPromptsGet:
		var params protocol.GetPromptRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			logger.Error("failed to unmarshal params", "error", err)
			return nil, jsonrpc2.ErrInvalidParams
		}
		res, err := h.PromptHandler.Handle(cctx, req.Method, params)
		if err != nil {
			return nil, fmt.Errorf("failed to handle %s: %w", req.Method, err)
		}
		return res, nil
	case req.Method == protocol.MethodResourcesList:
		if h.ResourceHandler == nil {
			logger.Error("resources/list is not supported")
			return nil, jsonrpc2.ErrMethodNotFound
		}

		cursor, err := nextCursorFromRequest(req)
		if err != nil {
			return nil, fmt.Errorf("failed to get next cursor: %w", err)
		}
		cctx = context.WithValue(cctx, nextCursorKey{}, cursor)

		res, err := h.ResourceHandler.HandleResourcesList(cctx)
		if err != nil {
			return nil, fmt.Errorf("failed to handle %s: %w", req.Method, err)
		}
		return res, nil
	case req.Method == protocol.MethodResourcesRead:
		if h.ResourceHandler == nil {
			logger.Error("resources/read is not supported")
			return nil, jsonrpc2.ErrMethodNotFound
		}
		var params ReadResourceRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			logger.Error("failed to unmarshal params", "error", err)
			return nil, jsonrpc2.ErrInvalidParams
		}
		res, err := h.ResourceHandler.HandleResourcesRead(cctx, &params)
		if err != nil {
			return nil, fmt.Errorf("failed to handle %s: %w", req.Method, err)
		}
		return res, nil
	case req.Method == protocol.MethodResourceTemplatesList:
		if h.Capabilities.Resources == nil {
			logger.Error("resources/templates/list is not supported")
			return nil, jsonrpc2.ErrMethodNotFound
		}

		return &listResourceTemplatesResult{
			ResourceTemplates: h.ResourceTemplates,
		}, nil
	case req.Method == protocol.MethodResourcesSubscribe:
		var params subscribeResourceRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			logger.Error("failed to unmarshal params", "error", err)
			return nil, jsonrpc2.ErrInvalidParams
		}
		h.subscribedResources.Store(params.URI, struct{}{})

		return struct{}{}, nil
	case req.Method == protocol.MethodResourcesUnsubscribe:
		var params unsubscribeResourceRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			logger.Error("failed to unmarshal params", "error", err)
			return nil, jsonrpc2.ErrInvalidParams
		}
		h.subscribedResources.Delete(params.URI)

		return struct{}{}, nil
	case req.Method == protocol.MethodToolsList:
		if h.Capabilities.Tools == nil {
			logger.Error("tools/list is not supported")
			return nil, jsonrpc2.ErrMethodNotFound
		}
		return &listToolsResult{
			Tools: h.Tools,
		}, nil
	case req.Method == protocol.MethodToolsCall:
		var params protocol.CallToolRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			logger.Error("failed to unmarshal params", "error", err)
			return nil, jsonrpc2.ErrInvalidParams
		}

		res, err := h.ToolHandler.Handle(cctx, req.Method, params)
		if err != nil {
			return nil, fmt.Errorf("failed to handle %s: %w", req.Method, err)
		}
		return res, nil
	case req.Method == protocol.MethodLoggingSetLevel:
		var params protocol.LoggingSetLevelRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			logger.Error("failed to unmarshal params", "error", err)
			return nil, jsonrpc2.ErrInvalidParams
		}
		minimumLogLevel.Set(slog.Level(params.Level))
		return struct{}{}, nil
	case req.Method == protocol.MethodNotificationsCancelled:
		var params protocol.NotificationsCancelledRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			logger.Error("failed to unmarshal params", "error", err)
			return nil, jsonrpc2.ErrInvalidParams
		}
		id := fmt.Sprintf("%v", params.RequestID)
		v, ok := h.cancelFuncByRequestID.LoadAndDelete(id)
		if ok {
			cancelFunc := v.(context.CancelFunc)
			cancelFunc()
		}
		return nil, nil
	case req.Method == protocol.MethodCompletionComplete:
		var params CompleteRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			logger.Error("failed to unmarshal params", "error", err)
			return nil, jsonrpc2.ErrInvalidParams
		}
		if h.CompletionHandler == nil {
			logger.Error("completion/complete is not supported")
			return nil, jsonrpc2.ErrMethodNotFound
		}
		res, err := h.CompletionHandler.HandleComplete(cctx, &params)
		if err != nil {
			return nil, fmt.Errorf("failed to handle %s: %w", req.Method, err)
		}
		return struct {
			Completion *CompleteResult `json:"completion"`
		}{
			Completion: res,
		}, nil
	default:
		logger.Error("unknown method", "method", req.Method)
		return nil, jsonrpc2.ErrMethodNotFound
	}
}

// IsSubscribed checks if the given resource is subscribed.
func (h *Handler) IsSubscribed(uri string) bool {
	_, ok := h.subscribedResources.Load(uri)
	return ok
}

type stdio struct {
	in  io.ReadCloser
	out io.WriteCloser
}

func (s stdio) Read(p []byte) (n int, err error)  { return s.in.Read(p) }
func (s stdio) Write(p []byte) (n int, err error) { return s.out.Write(p) }
func (s stdio) Close() error                      { return errors.Join(s.in.Close(), s.out.Close()) }

type stdioDialer struct{ stdio }

func (d *stdioDialer) Dial(ctx context.Context) (io.ReadWriteCloser, error) { return d.stdio, nil }

type stdioListener struct {
	stdio

	tokens chan struct{}
}

type stdioCloser struct {
	stdio
	close func() error
}

func (l *stdioListener) Accept(ctx context.Context) (io.ReadWriteCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case l.tokens <- struct{}{}:
	}

	return &stdioCloser{stdio: l.stdio, close: func() error {
		<-l.tokens
		return l.stdio.Close()
	}}, nil
}

func (l *stdioListener) Dialer() jsonrpc2.Dialer { return &stdioDialer{stdio: l.stdio} }

type framer struct {
	jsonrpc2.Framer
}

func (f *framer) Writer(rw io.Writer) jsonrpc2.Writer {
	writer := f.Framer.Writer(rw)
	return &framerWriter{Writer: writer, rw: rw}
}

type framerWriter struct {
	jsonrpc2.Writer
	rw io.Writer
}

func (w *framerWriter) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	n, err := w.Writer.Write(ctx, msg)
	if err != nil {
		return 0, err
	}
	_, err = w.rw.Write([]byte("\n"))
	if err != nil {
		return 0, err
	}
	return n, nil
}

// binder is an implementation of jsonrpc2.Binder
type binder struct {
	handler   jsonrpc2.Handler
	preempter jsonrpc2.Preempter
}

func (b *binder) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	return jsonrpc2.ConnectionOptions{
		Framer:    &framer{Framer: jsonrpc2.RawFramer()},
		Preempter: b.preempter,
		Handler:   b.handler,
	}, nil
}

type StdioTransportOptions struct {
	// MaxConns is the maximum number of connections that can be handled by the transport.
	// If this is not set, 5 connections are allowed.
	MaxConns int
	// Preempter is the preempter for the transport.
	// If this is not set, no preemption is done.
	Preempter jsonrpc2.Preempter
}

// NewStdioTransport creates a new stdio transport.
//
// See https://modelcontextprotocol.io/specification/2025-03-26/server/utilities/stdio#stdio
func NewStdioTransport(
	ctx context.Context,
	handler *Handler,
	opts *StdioTransportOptions,
) (context.Context, jsonrpc2.Listener, jsonrpc2.Binder) {
	w := io.Discard
	if handler.Capabilities.Logging != nil {
		w = os.Stdout
	}
	ctx = SetLogWriterToContext(ctx, w)

	if opts == nil {
		opts = &StdioTransportOptions{}
	}
	if opts.MaxConns == 0 {
		opts.MaxConns = 5
	}

	listener := &stdioListener{
		stdio:  stdio{in: os.Stdin, out: os.Stdout},
		tokens: make(chan struct{}, opts.MaxConns),
	}
	binder := &binder{handler: handler, preempter: opts.Preempter}

	return ctx, listener, binder
}

// logRecord represents a log record to be sent as a notification.
type logRecord struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

// logHandler struct manages logging for MCP
type logHandler struct {
	slog.Handler

	name string

	mu      *sync.Mutex
	encoder *json.Encoder
	buf     *bytes.Buffer
}

func (s *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	new := *s
	new.Handler = s.Handler.WithAttrs(attrs)
	return &new
}

func (s *logHandler) WithGroup(name string) slog.Handler {
	new := *s
	new.Handler = s.Handler.WithGroup(name)
	return &new
}

func (s *logHandler) Handle(ctx context.Context, r slog.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Handler.Handle(ctx, r); err != nil {
		return fmt.Errorf("failed to handle log: %w", err)
	}
	data := s.buf.String()
	s.buf.Reset()

	return s.encoder.Encode(logRecord{
		JSONRPC: "2.0",
		Method:  "notifications/message",
		Params: map[string]any{
			"level":  levelNameForLogging(r.Level),
			"logger": s.name,
			"data":   json.RawMessage(data),
		},
	})
}

// logWriterKey is a key for retrieving the log writer from the context
type logWriterKey struct{}

// SetLogWriterToContext sets the log writer to the context. This function is intended to be called by functions that creates a new transport.
func SetLogWriterToContext(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, logWriterKey{}, w)
}

// Logger creates a new logger with the given name.
// Note that this logger is for communication with the client, not for internal logging.
// The logged messages are sent as notifications to the client.
//
// See https://modelcontextprotocol.io/specification/2025-03-26/server/utilities/logging#logging
func Logger(ctx context.Context, name string) *slog.Logger {
	writer := ctx.Value(logWriterKey{}).(io.Writer)
	handler := newLogHandler(name, writer)
	return slog.New(handler)
}

// levelNameForLogging maps a slog level to a MCP logging level name.
func levelNameForLogging(level slog.Level) string {
	switch {
	case level <= slog.LevelDebug:
		return "debug"
	case level <= slog.LevelInfo:
		return "info"
	case level <= slog.Level(1): // Notice
		return "notice"
	case level <= slog.LevelWarn:
		return "warning"
	case level <= slog.LevelError:
		return "error"
	case level <= slog.Level(9): // Critical
		return "critical"
	case level <= slog.Level(10): // Alert
		return "alert"
	default:
		return "emergency"
	}
}

// newLogHandler creates a new log handler.
func newLogHandler(name string, w io.Writer) *logHandler {
	buf := &bytes.Buffer{}
	handler := &logHandler{
		name:    name,
		encoder: json.NewEncoder(w),
		buf:     buf,
		mu:      &sync.Mutex{},
		Handler: slog.NewJSONHandler(buf, &slog.HandlerOptions{
			Level: minimumLogLevel,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if len(groups) != 0 {
					return a
				}

				switch a.Key {
				case slog.TimeKey, slog.LevelKey, slog.SourceKey:
					return slog.Attr{}
				default:
					return a
				}
			},
		}),
	}
	return handler
}

// nextCursorKey is a key for retrieving the cursor value from the context
type nextCursorKey struct{}

// nextCursorFromRequest retrieves the cursor value from the request
func nextCursorFromRequest(req *jsonrpc2.Request) (string, error) {
	var p protocol.PaginationParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return "", fmt.Errorf("failed to unmarshal pagination params: %w", err)
	}
	return p.Cursor, nil
}

// NextCursor returns the next cursor from the context.
// If there is no next cursor or the API doesn't support pagination, it returns false.
func NextCursor(ctx context.Context) (string, bool) {
	v := ctx.Value(nextCursorKey{})
	if v == nil {
		return "", false
	}
	return v.(string), true
}
