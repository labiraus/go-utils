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
	add          bool
	connectionID uuid.UUID
	outbound     chan<- []byte
	path         string
}

type messageRequest struct {
	path    string
	message []byte
}

type InboundChan struct {
	ConnectionID uuid.UUID
	Inbound      <-chan []byte
}

const (
	inboundWS = iota
	outboundWS
	duplexWS
)

var (
	registrationChan = make(chan registration, 100)
	messageChan      = make(chan messageRequest, 100)
	upgrader         websocket.Upgrader
)

func InitOutbound(ctx context.Context, origin string) <-chan struct{} {
	done := make(chan struct{})
	if len(origin) > 0 {
		upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return r.Header.Get("Origin") == origin
			},
		}
	} else {
		upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
				// return r.Header.Get("Origin") == "https://your-allowed-origin.com"
			},
		}
	}

	go func() {
		defer close(done)
		registrations := make(map[string]map[uuid.UUID]chan<- []byte)
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
					slog.InfoContext(ctx, "listener connected", "connectionID", reg.connectionID, "path", reg.path)
					if _, ok := registrations[reg.path]; !ok {
						registrations[reg.path] = make(map[uuid.UUID]chan<- []byte)
					}
					registrations[reg.path][reg.connectionID] = reg.outbound
				} else {
					slog.InfoContext(ctx, "listener disconnected", "connectionID", reg.connectionID, "path", reg.path)
					outbound := registrations[reg.path][reg.connectionID]
					if outbound != nil {
						close(outbound)
						delete(registrations[reg.path], reg.connectionID)
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

func ServeInbound(mux *http.ServeMux, path string) <-chan InboundChan {
	return serve(mux, path, inboundWS)
}

func ServeDuplex(mux *http.ServeMux, path string) <-chan InboundChan {
	return serve(mux, path, duplexWS)
}

func ServeOutbound(mux *http.ServeMux, path string) {
	serve(mux, path, outboundWS)
}

func serve(mux *http.ServeMux, path string, wsType int) <-chan InboundChan {
	output := make(chan InboundChan, 100)

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
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
		connectionID := uuid.New()
		slog.InfoContext(r.Context(), "recieved connection", "connectionID", connectionID)

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Could not upgrade to websocket", http.StatusInternalServerError)
			return
		}
		defer conn.Close()
		defer conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "goodbye"))

		done := make(chan struct{})
		if wsType == outboundWS {
			// Outbound websockets only need to detect if the client has disconnected
			go func() {
				for {
					if _, _, err := conn.ReadMessage(); err != nil {
						close(done)
						return
					}
				}
			}()

		} else {
			inbound := make(chan []byte, 100)
			output <- InboundChan{
				ConnectionID: connectionID,
				Inbound:      inbound,
			}

			go func() {
				for {
					_, message, err := conn.ReadMessage()
					if err != nil {
						close(inbound)
						close(done)
						return
					}
					inbound <- message
				}
			}()
		}

		if wsType == inboundWS {
			<-done
		} else {
			defer func() {
				registrationChan <- registration{
					add:          false,
					connectionID: connectionID,
				}
			}()
			outbound := make(chan []byte, 100)
			registrationChan <- registration{
				add:          true,
				connectionID: connectionID,
				outbound:     outbound,
				path:         r.URL.Path,
			}
			for {
				select {
				case msg, ok := <-outbound:
					if !ok {
						return
					}
					if err = conn.WriteMessage(websocket.BinaryMessage, []byte(msg)); err != nil {
						slog.InfoContext(r.Context(), err.Error())
						return
					}
				case <-done:
					return
				}
			}
		}
	})
	return output
}

func Push(message []byte, path string) {
	messageChan <- messageRequest{path: path, message: message}
}

func Inbound(ctx context.Context, url string) <-chan []byte {
	return connect(ctx, url, make(<-chan []byte), inboundWS)
}

func Duplex(ctx context.Context, url string, outbound <-chan []byte) <-chan []byte {
	return connect(ctx, url, outbound, duplexWS)
}

func Outbound(ctx context.Context, url string, outbound <-chan []byte) {
	connect(ctx, url, outbound, outboundWS)
}

func connect(ctx context.Context, url string, outbound <-chan []byte, wsType int) <-chan []byte {
	inbound := make(chan []byte, 100)

	go func() {
		defer close(inbound)
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			slog.ErrorContext(ctx, "failed to connect to websocket", "error", err, "url", url)
			return
		}
		defer conn.Close()
		go func() {
			defer conn.Close()
			<-ctx.Done()
		}()
		if wsType != inboundWS {
			go func() {
				// This will close the connection if there's a write error or if the outbound channel is closed
				defer conn.Close()
				defer conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "goodbye"))
				for msg := range outbound {
					if err := conn.WriteMessage(websocket.BinaryMessage, []byte(msg)); err != nil {
						break
					}
				}
			}()
		}

		if wsType == outboundWS {
			<-ctx.Done()
		} else {
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
				inbound <- message
			}
		}
	}()

	return inbound
}
