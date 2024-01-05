package intercom

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative intercom.proto

// Port is the port of the intercom API.
const Port = "7777"
