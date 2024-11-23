package main

import (
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/labiraus/go-utils/pkg/api"
	"github.com/labiraus/go-utils/pkg/base"
)

//go:embed html
var content embed.FS

func main() {
	var err error
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		if err != nil {
			slog.Error(err.Error())
		}
	}()
	ctx := base.Init("webserver")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(content))))
	done := api.Init(ctx)

	template.ParseFS(content, "*.tmpl")
	close(base.Ready)
	<-done
	slog.Info("finishing")
}
