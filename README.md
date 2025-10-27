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

### Running the application

```sh
# To create a server, open a new terminal and run
$ go run server/server.go

# To create a client, open a new terminal and run
$ go run client/client.go
```

### Shutting down

**Server:**
- Press `Ctrl+C` to gracefully shutdown the server
- The server will log the shutdown event and exit

**Client:**
- Press `Ctrl+C` to disconnect and exit
- Type `LEAVE` (all caps) and press Enter to gracefully leave the chat and exit

Both methods will trigger proper cleanup and logging.

### Troubleshooting

If you need to stop old server instances you can use:

```sh
# Find processes on port 5050
lsof -ti:5050

# Kill them
kill -9 $(lsof -ti:5050)
```
