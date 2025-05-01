package main

import "fmt"

var data = map[int]string{}
var requests = make(chan request)

type request struct {
	read     bool
	id       int
	value    string
	response chan string
}

func main() {
	go actor(requests)

	go handler()
	go reader()
}

func handler() {
	requests <- request{
		id:    1,
		value: "hi",
		read:  false,
	}
}

func reader() {
	response := make(chan string, 1)
	requests <- request{
		id:       1,
		read:     true,
		response: response,
	}
	fmt.Println(<-response)
}

func actor(requests chan request) {
	for req := range requests {
		if req.read {
			req.response <- data[req.id]
		} else {
			data[req.id] = req.value
		}
	}
}
