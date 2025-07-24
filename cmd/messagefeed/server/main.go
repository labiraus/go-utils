package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/labiraus/go-utils/cmd/messagefeed/types"
	"github.com/labiraus/go-utils/pkg/api"
	"github.com/labiraus/go-utils/pkg/base"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type registration struct {
	add        bool
	listenerID uuid.UUID
	outbound   chan<- string
}

type messageRequest struct {
	UserID  string `json:"userid"`
	Message string `json:"message"`
}

var (
	registrationChan = make(chan registration, 100)
	messageChan      = make(chan messageRequest, 100)

	host     = "localhost"
	client   types.StoreClient
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
			// return r.Header.Get("Origin") == "https://your-allowed-origin.com"
		},
	}
)

func main() {
	var err error
	ctx := base.Init("messagefeed-server")
	defer func() {
		p := recover()
		if p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
		if err != nil {
			slog.ErrorContext(ctx, err.Error())
		}
	}()

	grpcPort := flag.Int("grpc", 50051, "the gRPC port to write to")
	port := flag.Int("port", 8080, "the HTTP port to listen to")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/listen", webSocketHandler)
	mux.HandleFunc("/post", messageHandler)
	api.Init(ctx, mux, *port)
	conn, err := grpc.NewClient(fmt.Sprintf("%v:%d", host, *grpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return
	}
	client = types.NewStoreClient(conn)
	done := actor(ctx)

	<-done
}

func webSocketHandler(w http.ResponseWriter, r *http.Request) {
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
	messageList, err := client.GetLast10(r.Context(), &types.Empty{})
	if err != nil {
		http.Error(w, "Could not get last 10 messages", http.StatusInternalServerError)
		return
	}

	for _, message := range messageList.Messages {
		outbound <- message.UserId + ":" + message.Message
	}

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
	}

	// This will close the connection if there's a write error or if the outbound channel is closed
	defer conn.Close()
	defer conn.WriteMessage(websocket.TextMessage, []byte("goodbye"))
	for msg := range outbound {
		if err = conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			break
		}
	}
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
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

	slog.InfoContext(r.Context(), "messageHandler called")

	var request = messageRequest{}
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

	messageChan <- request
}

func actor(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		registrations := make(map[uuid.UUID]chan<- string)
		for {
			select {
			case <-ctx.Done():
				for _, reg := range registrations {
					close(reg)
				}
				return
			case reg := <-registrationChan:
				if reg.add {
					slog.InfoContext(ctx, "listener connected", "listenerID", reg.listenerID)
					registrations[reg.listenerID] = reg.outbound
				} else {
					slog.InfoContext(ctx, "listener disconnected", "listenerID", reg.listenerID)
					delete(registrations, reg.listenerID)
					close(reg.outbound)
				}
			case message := <-messageChan:
				client.Save(ctx, &types.Message{UserId: message.UserID, Message: message.Message})
				for _, reg := range registrations {
					reg <- message.UserID + ":" + message.Message
				}
			}
		}
	}()

	return done
}
