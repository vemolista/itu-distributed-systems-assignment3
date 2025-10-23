# Distributed Systems - Fall 2025 - Assignment 3 - Chit Chat

## Contributing

### Generating protobuf code

Prerequisites (first time only):

```sh
# Install the Go plugins for protoc
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure Go bin is on PATH so protoc can find the plugins
export PATH="$(go env GOPATH)/bin:$PATH"
```

Generate code:

```sh
protoc --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  grpc/proto.proto
```

Running the application
 
```sh
# To create a client, open a new terminal and run
$ go run client/client.go

# To create a server, open a new terminal and run
$ go run server/server.go
```
