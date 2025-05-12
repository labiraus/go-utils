package main

import (
	"fmt"
	"net/http"
)

// main -> handlers, Actor
func main() {
	go actor()
}

func testFunc() {
	go func() {
		<-Requests
		fmt.Println("got request")
	}()

	save(item{})
}

// Handlers -> savelogic, actor
func saveHandler(w http.ResponseWriter, r *http.Request)   {}
func loadHandler(w http.ResponseWriter, r *http.Request)   {}
func deleteHandler(w http.ResponseWriter, r *http.Request) {}

// Save logic -> actor
func save(i item)   { Requests <- request{} }
func load() item    { return item{} }
func delete(id int) {}

// Actor
type item struct{}
type request struct{}

var Requests = make(chan request, 10)

func actor() {
	store := map[int]item{}
	// iterate over a channel
	for req := range Requests {

	}
}
