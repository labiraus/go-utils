package todo

import (
	"context"
	"log/slog"
)

var buffer chan request

const (
	getRequest    = "get"
	putRequest    = "put"
	deleteRequest = "delete"
)

type request struct {
	user        User
	requestType string
	response    chan []Item
	item        Item
}

func Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	buffer = make(chan request, 100)

	go func() {
		defer close(buffer)
		<-ctx.Done()
	}()

	go func() {
		defer close(done)
		// store := make(map[int]map[int]Item)
		store := make(map[int][]Item)
		for req := range buffer {
			switch req.requestType {
			case getRequest:
				data, ok := store[req.user.UserID]
				if ok {
					req.response <- data
				} else {
					slog.Warn("user not found")
				}
			case putRequest:
				data, ok := store[req.user.UserID]
				if !ok {
					data = make([]Item, 0)
				}
				data = append(data, req.item)
				store[req.user.UserID] = data

			case deleteRequest:
				_, ok := store[req.user.UserID]
				if ok {
					slog.Warn("delete not implemented")
				}
			}
			close(req.response)
		}
	}()
	return done
}

func Get(user User) []Item {
	response := make(chan []Item, 1)
	req := request{
		requestType: getRequest,
		user:        user,
		response:    response,
	}
	buffer <- req
	return <-response
}

func Put(user User, item Item) {
	response := make(chan []Item, 1)
	req := request{
		requestType: putRequest,
		user:        user,
		item:        item,
		response:    response,
	}
	buffer <- req
	<-response
}

func Delete(user User, item Item) {
	response := make(chan []Item, 1)
	req := request{
		requestType: getRequest,
		user:        user,
		item:        item,
		response:    response,
	}
	buffer <- req
	<-response
}
