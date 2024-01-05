package coordapi

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative coordapi.proto

// Port is the port of the coordinator API.
const Port = "1313"
