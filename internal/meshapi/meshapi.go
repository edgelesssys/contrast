package meshapi

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative meshapi.proto

// Port is the port of the mesh API.
const Port = "7777"
