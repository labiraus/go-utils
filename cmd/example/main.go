package main

import (
	"fmt"
)

const (
	read  = "read"
	write = "write"
)

type message struct {
	key          string
	data         string
	action       string
	responseChan chan<- string
}

var (
	readChan  = make(chan message, 100)
	writeChan = make(chan message, 100)
	i         = 0
)

func main() {
	go actor()
	go thread1()
	go thread2()
}

func thread1() {
	done := make(chan string)
	writeChan <- message{
		key:          "hi",
		data:         "ho",
		action:       write,
		responseChan: done,
	}
	<-done
}

func thread2() {
	output := make(chan string)
	readChan <- message{
		key:          "hi",
		action:       read,
		responseChan: output,
	}

	fmt.Println(<-output)
}

func actor() {
	data := map[string]string{}
	for {
		select {
		case msg := <-writeChan:
			data[msg.key] = msg.data
			continue
		default:
		}

		select {
		case msg := <-readChan:
			msg.responseChan <- data[msg.key]
		case msg := <-writeChan:
			data[msg.key] = msg.data
		}
	}

	for msg := range messageChan {
		switch msg.action {
		case read:
			msg.responseChan <- data[msg.key]
		case write:
			data[msg.key] = msg.data
		}
		close(msg.responseChan)
	}
}
