package main

import (
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/labiraus/go-utils/pkg/api"
	"github.com/labiraus/go-utils/pkg/base"
)

var (
	//go:embed static
	static embed.FS
	//go:embed dynamic
	dynamic embed.FS
	tmpl    *template.Template
)

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

	tmpl, err = template.ParseFS(dynamic, "dynamic/*.tmpl")
	if err != nil {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveDynamic)
	mux.Handle("/static/", http.FileServer(http.FS(static)))
	done := api.Init(ctx, mux)

	close(base.Ready)
	<-done
	slog.Info("finishing")
}

type PageData struct {
	Title string
	Year  int
}

func serveDynamic(w http.ResponseWriter, r *http.Request) {
	data := PageData{Title: "Dynamic Content", Year: time.Now().Year()}
	tmpl.Execute(w, data)
}
