package base

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/google/uuid"
)

const TraceIDString = "trace_id"

var (
	Ready       = make(chan struct{})
	ServiceName string
)

type customHandler struct {
	slog.Handler
}

func (h *customHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceID, ok := ctx.Value(TraceIDString).(string); ok {
		r.AddAttrs(slog.String(TraceIDString, traceID))
	}
	if traceID, ok := ctx.Value(TraceIDString).(uuid.UUID); ok {
		r.AddAttrs(slog.String(TraceIDString, traceID.String()))
	}
	return h.Handler.Handle(ctx, r)
}

func Init(serviceName string) context.Context {
	ServiceName = serviceName
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
	handler := &customHandler{Handler: baseHandler.WithGroup(serviceName)}
	logger := slog.New(handler)
	slog.SetDefault(logger)

	ctx, ctxDone := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, TraceIDString, uuid.New())
	slog.InfoContext(ctx, "starting")
	go func() {
		defer ctxDone()
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		s := <-c
		slog.InfoContext(ctx, "got signal: ["+s.String()+"] now closing")
	}()

	go func() {
		<-Ready
		slog.InfoContext(ctx, "ready")
	}()

	return ctx
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
