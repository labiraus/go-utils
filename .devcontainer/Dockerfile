FROM mcr.microsoft.com/devcontainers/go:1-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    curl \
    bash-completion \
    protobuf-compiler && \
    rm -rf /var/lib/apt/lists/*

COPY zscaler.crt /usr/local/share/ca-certificates/zscaler.crt
RUN update-ca-certificates

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN go install golang.org/x/tools/cmd/stringer@latest

RUN chown -R vscode:vscode /go/pkg/mod
RUN mkdir -p /go/pkg/sumdb && chown -R vscode:vscode /go/pkg/sumdb /go/pkg/mod
