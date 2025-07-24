#!/bin/bash

# Path to the types.go file
current_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
types_file=$current_dir"/types.proto"

# Generate protobuf files
protoc -I=$current_dir \
  --go_out=$current_dir --go_opt=paths=source_relative \
  --go-grpc_out=$current_dir --go-grpc_opt=paths=source_relative \
  $types_file

# Uncomment the following line if you need to specify options for the generated files
# protoc -I=$output_dir --go_out=$output_dir --go_opt=Mtypes.proto=utils/types --go-grpc_out=$output_dir --go-grpc_opt=Mtypes.proto=utils/types $types_file