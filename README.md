# github-access-policies

## Generating gRPC Files

To generate new gRPC files from the proto definitions, you'll need to have the Protocol Buffers compiler (`protoc`) and the Go gRPC plugin installed.

### Prerequisites

1. Install Protocol Buffers compiler:

   ```bash
   # On macOS with Homebrew
   brew install protobuf

   # On Ubuntu/Debian
   apt-get install protobuf-compiler
   ```

2. Install Go plugins for protoc:
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

### Generate Files

Run the following command from the project root to generate Go files from the proto definitions:

```bash
protoc --go_out=./gen --go_opt=paths=source_relative \
       --go-grpc_out=./gen --go-grpc_opt=paths=source_relative \
       api/github_scanner.proto
```

This will generate:

- `gen/github_scanner.pb.go` - Protocol Buffer message definitions
- `gen/github_scanner_grpc.pb.go` - gRPC service definitions

### Note

After generating new files, make sure to run:

```bash
go mod tidy
```

This ensures all dependencies are properly resolved and added to your `go.mod` file.
