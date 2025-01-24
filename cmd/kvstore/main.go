package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"

	"github.com/labiraus/go-utils/pkg/api"
	"github.com/labiraus/go-utils/pkg/base"
)

var requestBuffer chan<- apiRequest

type apiRequest struct {
	verb     string
	key      string
	value    []byte
	response chan<- []byte
}

func main() {
	ctx := base.Init("kvstore")
	mux := http.NewServeMux()
	done := startApi(ctx, mux)
	api.Init(ctx, mux)

	close(base.Ready)
	<-done
}

func startApi(ctx context.Context, mux *http.ServeMux) <-chan struct{} {
	requests := make(chan apiRequest, 10)
	requestBuffer = requests
	mux.HandleFunc("/", handle)
	done := actor(ctx, requests)

	return done
}

func actor(ctx context.Context, requests chan apiRequest) <-chan struct{} {
	done := make(chan struct{})
	// Shutdown
	go func() {
		<-ctx.Done()
		close(requests)
	}()

	go func() {
		defer close(done)
		<-processLoop(requests)
	}()

	return done
}

func processLoop(requests <-chan apiRequest) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		store := make(map[string][]byte)
		defer close(done)
		for req := range requests {
			switch req.verb {
			case http.MethodDelete:
				delete(store, req.key)

			case http.MethodPut:
			case http.MethodPost:
			case http.MethodPatch:
				store[req.key] = req.value

			case http.MethodGet:
				value, ok := store[req.key]
				if !ok {
					log.Printf("could not find value for key %v", req.key)
				} else {
					req.response <- value
				}
			}
			close(req.response)
		}
	}()
	return done
}

func handle(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		p := recover()
		if p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
		if err != nil {
			slog.ErrorContext(r.Context(), err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	responseChan := make(chan []byte, 1)
	request := apiRequest{verb: r.Method, key: r.URL.Path, response: responseChan}
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		body, err := io.ReadAll(io.Reader(r.Body))
		defer r.Body.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		request.value = body
	}

	select {
	case requestBuffer <- request:
	default:
		slog.WarnContext(r.Context(), "request buffer full or shutdown")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	response, ok := <-responseChan

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusOK)
		return
	}

	if ok {
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
