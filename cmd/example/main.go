package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	itemList := loadTodos()
	action := flag.String("action", "", "Action to make")
	todoItem := flag.String("itemName", "", "Item to add")
	todoDesc := flag.String("itemDesc", "", "Description of Item")

	flag.Parse()

	if *todoItem == "" {
		fmt.Println("Missing itemName")
		return
	}

	switch *action {
	case "add":
		if *todoDesc == "" {
			fmt.Println("Missing itemDesc for add")
			return
		}
		itemList[*todoItem] = *todoDesc

	case "update":
		if *todoDesc == "" {
			fmt.Println("Missing itemDesc for update")
			return
		}
		if _, ok := itemList[*todoItem]; !ok {
			fmt.Println(*todoItem, "doesn't exist")
			return
		}
		itemList[*todoItem] = *todoDesc

	case "delete":
		delete(itemList, *todoItem)

	default:
		fmt.Println("Invalid action")
		return
	}
	fmt.Printf("%q", itemList)
	saveToDo(itemList)
}

func loadTodos() map[string]string {
	itemList := make(map[string]string)
	f, err := os.Open("todo.txt")
	if err != nil {
		log.Fatalf("failed file open")
	}
	defer f.Close()
	json.NewDecoder(f).Decode(&itemList)
	return itemList
}

func saveToDo(itemList map[string]string) {
	f, err := os.Create("todo.txt")
	if err != nil {
		log.Fatalf("failed file create")
	}
	defer f.Close()
	json.NewEncoder(f).Encode(itemList)

}
