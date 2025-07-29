package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/labiraus/go-utils/pkg/base"
	"github.com/labiraus/go-utils/pkg/repl"
)

var name string

var rootMenu []repl.CliOption
var done = make(chan struct{})

func main() {
	rootMenu = []repl.CliOption{
		{Key: "yes", Description: "agree with me", Action: yes},
		{Key: "no", Description: "disagree with me", Action: no},
	}
	ctx := base.Start("pubsubrepl")
	close(base.Ready)
	repl.StartReading(ctx)
	fmt.Println("What's your name?")
	name = repl.Read(ctx)
	repl.PresentOptions(ctx, rootMenu...)
	select {
	case <-ctx.Done():
	case <-done:
	}
	slog.Info("finishing")
}

func yes(ctx context.Context) {
	fmt.Println("You have excellent taste", name)
	repl.PresentOptions(ctx, append([]repl.CliOption{{Key: "rename", Description: "rename yourself", Action: rename}}, rootMenu...)...)
}

func no(ctx context.Context) {
	fmt.Println("You suck", name)
	repl.PresentOptions(ctx, append([]repl.CliOption{
		{Key: "exit", Description: "quit out", Action: quit}},
		rootMenu...)...)
}

func rename(ctx context.Context) {
	fmt.Println("What's your name?")
	name = repl.Read(ctx)
	repl.PresentOptions(ctx, rootMenu...)
}

func quit(ctx context.Context) {
	close(done)
}
