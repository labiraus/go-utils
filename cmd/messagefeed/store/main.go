package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/labiraus/go-utils/cmd/messagefeed/types"
	"github.com/labiraus/go-utils/pkg/api"
	"github.com/labiraus/go-utils/pkg/base"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflections"
)

func main() {
	ctx := base.Start("messagefeed-store")

	grpcPort := flag.Int("grpc", 50051, "the GRPC port to listen on")
	flag.Parse()

	mux := http.NewServeMux()
	// liveliness and readiness need to be exposed regardless
	done := api.Start(ctx, mux, 8081)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	types.RegisterStoreServer(s, &store{})
	reflections.Register(s)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	<-done
}

type store struct {
	types.UnimplementedStoreServer
}

const filename = "todo.txt"

func (*store) Save(ctx context.Context, in *types.Message) (*types.Empty, error) {
	f, err := os.OpenFile("todo.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("failed file create")
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(map[string]string{in.UserId: in.Message})
	if err != nil {
		return &types.Empty{}, err
	}

	return &types.Empty{}, nil
}

func countLines() (int, error) {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // treat missing file as zero lines
		}
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, 8192)
	count := 0
	for {
		n, err := f.Read(buf)
		if n > 0 {
			for _, b := range buf[:n] {
				if b == '\n' {
					count++
				}
			}
		}
		if err == os.ErrClosed || err == io.EOF {
			break
		}
	}
	return count, err
}

func (*store) GetLast10(ctx context.Context, in *types.Empty) (*types.MessageList, error) {
	lineCount, err := countLines()
	if err != nil {
		slog.ErrorContext(ctx, "failed to count lines", "error", err.Error())
		return nil, err
	}

	output := types.MessageList{}
	if lineCount == 0 {
		return &output, nil
	}

	// First pass: count lines
	f, err := os.Open(filename)
	if err != nil {
		slog.ErrorContext(ctx, "failed to open file", "error", err.Error())
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	skip := 0
	if lineCount > 10 {
		skip = lineCount - 10
	}

	current := 0
	for scanner.Scan() {
		current++
		if current <= skip {
			continue
		}
		var entry map[string]string
		if err := json.Unmarshal([]byte(scanner.Text()), &entry); err == nil {
			for userID, message := range entry {
				output.Messages = append(output.Messages, &types.Message{UserId: userID, Message: message})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.ErrorContext(ctx, "failed to open file", "error", err.Error())
		return nil, err
	}
	return &output, nil
}
