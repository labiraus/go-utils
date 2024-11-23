package repl

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

type CliOption struct {
	Key         string
	Action      func(context.Context)
	Description string
}

var inputChan = make(chan string)

func StartReading(ctx context.Context, options ...CliOption) {
	scanner := bufio.NewScanner(os.Stdin)

	go func() {
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintln(os.Stderr, "Error reading input:", err)
				continue
			}
			select {
			case inputChan <- scanner.Text():
			case <-ctx.Done():
				fmt.Println("Input cancelled.")
				return
			}
		}
	}()
}

func PresentOptions(ctx context.Context, options ...CliOption) {
	fmt.Println("Options:")
	optionMap := make(map[string]func(context.Context), len(options))
	for _, value := range options {
		fmt.Printf("%s: %s\n", value.Key, value.Description)
		optionMap[value.Key] = value.Action
	}
	for {
		select {
		case <-ctx.Done():
			return
		case input := <-inputChan:
			input = strings.TrimSpace(strings.ToLower(input))
			if action, ok := optionMap[input]; ok {
				action(ctx)
				return
			}
			fmt.Println("Invalid option")
		}
	}
}

func Read(ctx context.Context) string {
	select {
	case <-ctx.Done():
		return ""
	case input := <-inputChan:
		return input
	}
}
