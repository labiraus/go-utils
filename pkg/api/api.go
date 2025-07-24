package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labiraus/go-utils/pkg/base"
)

func Init(ctx context.Context, mux *http.ServeMux, port int) <-chan struct{} {
	mux.HandleFunc("/readiness", readinessHandler)
	mux.HandleFunc("/liveness", livelinessHandler)

	done := make(chan struct{})
	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: contextMiddleware(ctx, traceIDMiddleware(mux)),
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			panic("ListenAndServe: " + err.Error())
		}
	}()

	go func() {
		defer close(done)

		<-ctx.Done()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("Shutdown: " + err.Error())
		}
	}()
	return done
}

func traceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceIDHeader := http.CanonicalHeaderKey(base.TraceIDString)
		traceID := r.Header[traceIDHeader]
		if len(traceID) != 0 {
			r = r.WithContext(context.WithValue(r.Context(), base.TraceIDString, traceID[0]))
		} else {
			r = r.WithContext(context.WithValue(r.Context(), base.TraceIDString, uuid.New().String()))
		}
		next.ServeHTTP(w, r)
	})
}

func contextMiddleware(ctx context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx, cancel := context.WithCancel(r.Context())
		context.AfterFunc(ctx, cancel)
		next.ServeHTTP(w, r.WithContext(rctx))
	})
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	<-base.Ready
	w.WriteHeader(http.StatusOK)
}

func livelinessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
