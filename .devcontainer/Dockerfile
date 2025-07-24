FROM mcr.microsoft.com/devcontainers/go:latest
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        curl \
        protobuf-compiler
        
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

RUN chown -R vscode:vscode /go/pkg/mod
RUN mkdir -p /go/pkg/sumdb && chown -R vscode:vscode /go/pkg/sumdb /go/pkg/mod
