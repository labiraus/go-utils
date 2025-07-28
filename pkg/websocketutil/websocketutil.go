package websocketutil

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type registration struct {
	add        bool
	listenerID uuid.UUID
	outbound   chan<- string
	path       string
}

type messageRequest struct {
	path    string
	message string
}

var (
	registrationChan = make(chan registration, 100)
	messageChan      = make(chan messageRequest, 100)
	upgrader         = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
			// return r.Header.Get("Origin") == "https://your-allowed-origin.com"
		},
	}
)

func Init(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		registrations := make(map[string]map[uuid.UUID]chan<- string)
		for {
			select {
			case <-ctx.Done():
				for _, regPath := range registrations {
					for _, reg := range regPath {
						close(reg)
					}
				}
				return
			case reg := <-registrationChan:
				if reg.add {
					slog.InfoContext(ctx, "listener connected", "listenerID", reg.listenerID, "path", reg.path)
					if _, ok := registrations[reg.path]; !ok {
						registrations[reg.path] = make(map[uuid.UUID]chan<- string)
					}
					registrations[reg.path][reg.listenerID] = reg.outbound
				} else {
					slog.InfoContext(ctx, "listener disconnected", "listenerID", reg.listenerID, "path", reg.path)
					outbound := registrations[reg.path][reg.listenerID]
					if outbound != nil {
						close(outbound)
						delete(registrations[reg.path], reg.listenerID)
					}
				}
			case message := <-messageChan:
				for _, reg := range registrations[message.path] {
					reg <- message.message
				}
			}
		}
	}()

	return done
}

func Register(mux *http.ServeMux, path string) {
	mux.HandleFunc(path, subscribeHandler)
}

func Push(message string, path string) {
	messageChan <- messageRequest{path: path, message: message}
}

func subscribeHandler(w http.ResponseWriter, r *http.Request) {
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
	listenerID := uuid.New()
	slog.InfoContext(r.Context(), "recieved connection", "listenerID", listenerID)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade to websocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	outbound := make(chan string, 100)
	done := make(chan struct{})

	// Goroutine to detect client disconnect
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				close(done)
				return
			}
		}
	}()

	defer func() {
		registrationChan <- registration{
			add:        false,
			listenerID: listenerID,
		}
	}()
	registrationChan <- registration{
		add:        true,
		listenerID: listenerID,
		outbound:   outbound,
		path:       r.URL.Path,
	}

	// This will close the connection if there's a write error or if the outbound channel is closed
	defer conn.Close()
	defer conn.WriteMessage(websocket.TextMessage, []byte("goodbye"))
	for {
		select {
		case msg, ok := <-outbound:
			if !ok {
				return
			}
			if err = conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				slog.InfoContext(r.Context(), err.Error())
				return
			}
		case <-done:
			return
		}
	}
}
