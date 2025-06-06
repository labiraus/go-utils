package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func resetGlobals() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	registrationChan = make(chan registration, 100)
}

func TestCreateRoom(t *testing.T) {
	resetGlobals()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomName := "test-room"
	room := createRoom(roomName, ctx)

	// Test that the room is initialized correctly
	assert.NotNil(t, room.registrations)
	assert.NotNil(t, room.done)
	assert.Equal(t, roomName, room.roomName)

	// Test adding a user to the room
	userID := uuid.New()
	outbound := make(chan string, 10)
	inbound := make(chan chan<- chatMessage, 1)

	reg := registration{
		add:      true,
		roomName: roomName,
		userID:   userID,
		outbound: outbound,
		inbound:  inbound,
	}

	go func() {
		room.registrations <- reg
	}()

	select {
	case inboundChan := <-inbound:
		assert.NotNil(t, inboundChan)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for registration to be processed")
	}

	// Test removing a user from the room
	reg.add = false
	go func() {
		room.registrations <- reg
	}()

	select {
	case <-room.done:
		// Room should not close since there are no users
	case <-time.After(1 * time.Second):
		// No timeout expected
	}
}

func TestRoomController(t *testing.T) {
	resetGlobals()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := roomController(ctx)

	// Test that the controller initializes correctly
	assert.NotNil(t, done)

	// Test adding a registration
	userID := uuid.New()
	outbound := make(chan string, 10)
	inbound := make(chan chan<- chatMessage, 1)

	reg := registration{
		add:      true,
		roomName: "test-room",
		userID:   userID,
		outbound: outbound,
		inbound:  inbound,
	}

	go func() {
		registrationChan <- reg
	}()

	select {
	case inboundChan := <-inbound:
		assert.NotNil(t, inboundChan)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for registration to be processed")
	}

	// Test shutting down the controller
	cancel()
	select {
	case <-done:
		// Controller should shut down cleanly
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for controller to shut down")
	}
}

func TestWebsocketHandler(t *testing.T) {
	resetGlobals()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	roomController(ctx)
	server := httptest.NewServer(http.HandlerFunc(websocketHandler))
	defer server.Close()
	outbound := make(chan string, 10)
	inboundCarrier := make(chan<- chan<- chatMessage, 1)
	registrationChan <- registration{
		add:      true,
		userID:   uuid.Max,
		outbound: outbound,
		inbound:  inboundCarrier,
		roomName: "/test",
	}
	// Convert the test server URL to a WebSocket URL
	wsURL := "ws" + server.URL[len("http"):] + "/test"

	// Connect to the WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Test sending a message
	err = conn.WriteMessage(websocket.TextMessage, []byte("hello"))
	assert.NoError(t, err)

	// Test receiving a message
	message := <-outbound
	assert.Equal(t, "hello", string(message))
}
