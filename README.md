# Distributed Systems - Fall 2025 - Assignment 3 - Chit Chat

## Contributing
Generating protobuf code
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

> [!CAUTION]
> The client's name is currently hardcoded to "john". Before creating more than one client is possible, we have to get rid of the username, and/or ask for it before the client attempts to join.