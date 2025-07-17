# github-access-policies

## File structure

- **/api** - This directory contains gRPC proto definitions and acts as our API schema.
  - **/v1** - Version 1 of the API definitions
  - **/v2** - Version 2 of the API definitions (future)
- **/gen** - This directory contains gRPC generated files from the /api directory.
  - **/v1** - Generated files for API version 1
  - **/v2** - Generated files for API version 2 (future)

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

You can generate files using either the `protoc` command directly
**For v1:**

```bash
protoc --go_out=./gen --go_opt=paths=source_relative \
  --go-grpc_out=./gen --go-grpc_opt=paths=source_relative \
  --proto_path=./api ./api/v1/github_scanner.proto
```

**For future v2:**

```bash
protoc --go_out=./gen --go_opt=paths=source_relative \
  --go-grpc_out=./gen --go-grpc_opt=paths=source_relative \
  --proto_path=./api ./api/v2/github_scanner.proto
```

This will generate:

- `gen/v1/github_scanner.pb.go` - Protocol Buffer message definitions (v1)
- `gen/v1/github_scanner_grpc.pb.go` - gRPC service definitions (v1)
- `gen/v2/github_scanner.pb.go` - Protocol Buffer message definitions (v2, when created)
- `gen/v2/github_scanner_grpc.pb.go` - gRPC service definitions (v2, when created)

### Note

After generating new files, make sure to run:

```bash
go mod tidy
```

This ensures all dependencies are properly resolved and added to your `go.mod` file.

## API Versioning Strategy

This project follows semantic versioning for its gRPC API:

- **v1**: Current stable API version
- **v2**: Future major version (breaking changes)

### Creating a New API Version

When you need to make breaking changes:

1. Create a new version directory:

   ```bash
   mkdir -p api/v2
   mkdir -p gen/v2
   ```

2. Copy the current proto file as a starting point:

   ```bash
   cp api/v1/github_scanner.proto api/v2/github_scanner.proto
   ```

3. Update the package name and go_package option in the new proto file:

   ```proto
   package githubscanner.v2;
   option go_package = "github.com/liadyacobi/github-access-policies/gen/v2";
   ```

4. Make your breaking changes to the v2 proto file

5. Generate the new version using the protoc command (shown above)

### Import Paths

When using the generated code in your Go applications:

```go
// For v1
import v1 "github.com/liadyacobi/github-access-policies/gen/v1"

// For v2 (when available)
import v2 "github.com/liadyacobi/github-access-policies/gen/v2"
```
