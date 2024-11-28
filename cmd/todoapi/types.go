package main

import "github.com/labiraus/go-utils/pkg/todo"

type postRequest struct {
	todo.User
	todo.Item
}
