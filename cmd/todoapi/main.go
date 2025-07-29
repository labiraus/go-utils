package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/labiraus/go-utils/pkg/api"
	"github.com/labiraus/go-utils/pkg/base"
	"github.com/labiraus/go-utils/pkg/todo"
)

func main() {
	var err error
	ctx := base.Start("todoapi")
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		if err != nil {
			slog.ErrorContext(ctx, err.Error())
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /todo", postHandler)
	mux.HandleFunc("GET /todo", getHandler)
	mux.HandleFunc("DELETE /todo", deleteHandler)

	done := todo.Start(ctx)

	api.Start(ctx, mux, 8080)

	close(base.Ready)
	<-done
	slog.InfoContext(ctx, "finishing")
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		p := recover()
		if p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
		if err != nil {
			slog.ErrorContext(r.Context(), err.Error())
		}
	}()

	slog.InfoContext(r.Context(), "post called")

	var request = postRequest{}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	todo.Put(request.User, request.Item)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		p := recover()
		if p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
		if err != nil {
			slog.ErrorContext(r.Context(), err.Error())
		}
	}()

	slog.InfoContext(r.Context(), "get called")

	var request = todo.User{}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	response := todo.Get(request)

	data, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		p := recover()
		if p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
		if err != nil {
			slog.ErrorContext(r.Context(), err.Error())
		}
	}()

	slog.InfoContext(r.Context(), "delete called")

	var request = postRequest{}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	todo.Delete(request.User, request.Item)
}
