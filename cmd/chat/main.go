package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labiraus/go-utils/pkg/api"
	"github.com/labiraus/go-utils/pkg/base"
)

type registration struct {
	add      bool
	roomName string
	userID   uuid.UUID
	outbound chan<- string
	inbound  chan<- chan<- string
}

type chatRoom struct {
	registrations chan registration
	done          chan struct{}
	roomName      string
}

var registrationChan = make(chan registration, 100)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
		// return r.Header.Get("Origin") == "https://your-allowed-origin.com"
	},
}

func main() {
	var err error
	ctx := base.Init("chat")
	defer func() {
		p := recover()
		if p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
		if err != nil {
			slog.ErrorContext(ctx, err.Error())
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("/", websocketHandler)
	done := roomController(ctx)
	api.Init(ctx, mux)

	<-done
}

func roomController(ctx context.Context) chan struct{} {
	done := make(chan struct{})

	// wg counts how many active rooms there are and allows all rooms to be shut down before
	var wg sync.WaitGroup
	go func() {
		defer close(done)
		<-ctx.Done()
		wg.Wait()
	}()

	go func() {
		roomRegistrations := map[string]chatRoom{}
		for reg := range registrationChan {
			room, ok := roomRegistrations[reg.roomName]
			if !ok {
				room = createRoom(reg.roomName, ctx)
				go func(room chatRoom) {
					defer wg.Done()
					defer delete(roomRegistrations, room.roomName)
					wg.Add(1)
					<-room.done
				}(room)
				roomRegistrations[reg.roomName] = room
			}

			go func() {
				// room registration is an unbuffered channel to prevent anyone from being added to a room whilst it's shutting down
				// this means that registration needs to be asynchronous
				select {
				case room.registrations <- reg:
				case <-room.done:
					close(reg.inbound)
				}
			}()
		}
	}()

	return done
}

func createRoom(room string, ctx context.Context) chatRoom {
	registrations := make(chan registration)
	done := make(chan struct{})

	go func() {
		defer close(done)

		users := map[uuid.UUID]chan<- string{}
		inbound := make(chan string, 1000)

		// the ticker kills empty channels after 10 sec
		ticker := time.NewTicker(10 * time.Second)

		for {
			select {
			case <-ctx.Done():
				for _, user := range users {
					close(user)
				}
				return

			case reg := <-registrations:
				if reg.add {
					users[reg.userID] = reg.outbound
					reg.inbound <- inbound
				} else {
					delete(users, reg.userID)
				}
				ticker.Reset(0)

			case message := <-inbound:
				for _, user := range users {
					user <- message
				}
				ticker.Reset(0)

			case <-ticker.C:
				if len(users) == 0 {
					close(registrations)
					slog.InfoContext(ctx, "cleanup", "room", room)
					return
				}
			}
		}
	}()

	return chatRoom{
		registrations: registrations,
		roomName:      room,
		done:          done,
	}
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade to websocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	userID, err := uuid.NewUUID()
	if err != nil {
		http.Error(w, "Failed to generate user ID", http.StatusInternalServerError)
		return
	}

	outbound := make(chan string, 100)
	inboundCarrier := make(chan chan<- string, 1)
	defer func() {
		registrationChan <- registration{
			add:      false,
			roomName: r.URL.Path,
			userID:   userID,
		}
	}()
	registrationChan <- registration{
		add:      true,
		roomName: r.URL.Path,
		userID:   userID,
		outbound: outbound,
		inbound:  inboundCarrier,
	}
	inbound, ok := <-inboundCarrier
	if !ok {
		http.Error(w, "Failed to join room", http.StatusInternalServerError)
		return
	}

	go func() {
		// This will close the connection if there's a write error or if the outbound channel is closed
		defer conn.Close()
		for msg := range outbound {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				break
			}
		}
		conn.WriteMessage(websocket.TextMessage, []byte("goodbye"))
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		strMessage := string(message)
		if strMessage != "" {
			inbound <- strMessage
		}
	}
}
