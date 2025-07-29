package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/gorilla/websocket"
	"github.com/labiraus/go-utils/pkg/base"
)

var port = flag.Int("port", 8080, "the HTTP port to listen to")

func main() {
	var err error
	ctx := base.Start("messagefeed-client")
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		if err != nil {
			slog.ErrorContext(ctx, err.Error())
		}
	}()

	flag.Parse()

	done := Listen(ctx)

	close(base.Ready)
	<-done
	slog.InfoContext(ctx, "finishing")
}

func Listen(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		url := fmt.Sprintf("ws://localhost:%d/listen", *port)
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			slog.ErrorContext(ctx, "failed to connect to websocket", "error", err)
			return
		}
		defer conn.Close()
		go func() {
			defer conn.Close()
			<-ctx.Done()
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				select {
				case <-ctx.Done():
				default:
					slog.ErrorContext(ctx, "read error", "error", err)
				}
				break
			}
			fmt.Fprintln(os.Stdout, string(message))
		}
	}()
	return done
}
