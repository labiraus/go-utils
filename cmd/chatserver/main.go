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

type chatMessage struct {
	text   string
	userID uuid.UUID
}

type registration struct {
	add      bool
	roomName string
	userID   uuid.UUID
	outbound chan<- string
	inbound  chan<- chan<- chatMessage
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
	ctx := base.Start("chat")
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
	api.Start(ctx, mux, 8080)

	<-done
}

func roomController(ctx context.Context) chan struct{} {
	done := make(chan struct{})
	roomRegistrations := map[string]chatRoom{}
	slog.InfoContext(ctx, "starting controller")

	// wg counts how many active rooms there are and allows all rooms to be shut down before
	var wg sync.WaitGroup
	go func() {
		defer close(done)
		<-ctx.Done()
		wg.Wait()
	}()

	go func() {
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

func createRoom(roomName string, ctx context.Context) chatRoom {
	slog.InfoContext(ctx, "creating room", "roomName", roomName)
	registrations := make(chan registration)
	done := make(chan struct{})

	go func() {
		defer close(done)

		users := map[uuid.UUID]chan<- string{}
		inbound := make(chan chatMessage, 1000)

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
					slog.InfoContext(ctx, "registering user", "userID", reg.userID, "roomName", reg.roomName)
				} else {
					user, ok := users[reg.userID]
					if ok {
						close(user)
					}
					slog.InfoContext(ctx, "deregistering user", "userID", reg.userID, "roomName", reg.roomName)
					delete(users, reg.userID)
				}
				ticker.Reset(10 * time.Second)

			case message := <-inbound:
				for userID, user := range users {
					if userID != message.userID {
						select {
						case user <- message.text:
						default:
							slog.ErrorContext(ctx, "unable to send messages to user, cutting them off", "userID", userID)
							close(user)
							delete(users, userID)
						}
					}
				}
				ticker.Reset(10 * time.Second)

			case <-ticker.C:
				if len(users) == 0 {
					close(registrations)
					slog.InfoContext(ctx, "cleanup", "room", roomName)
					return
				}
			}
		}
	}()

	return chatRoom{
		registrations: registrations,
		roomName:      roomName,
		done:          done,
	}
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	userID := uuid.New()
	slog.InfoContext(r.Context(), "recieved connection", "userID", userID)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade to websocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	outbound := make(chan string, 100)
	inboundCarrier := make(chan chan<- chatMessage, 1)
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
			inbound <- chatMessage{
				userID: userID,
				text:   strMessage,
			}
		}
	}
}
