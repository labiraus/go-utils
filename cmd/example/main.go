package main

import (
	"fmt"
)

type thing struct{
	printer func(string)
	message string
}

func main() {
	t := CreateThing2(func(msg string){
		fmt.Println(msg)
	})
	DoSomething(t)
}

func CreateThing2(p func(string))thing{
	return thing{printer = p}
}

func CreateThing(message string)thing{
	return CreateThing2(func(msg string){
		fmt.Println(msg)
	})
}

func (t *thing) Print(){
	fmt.Println(t.message)
}

func DoSomething(t thing){
	t.Print()
}