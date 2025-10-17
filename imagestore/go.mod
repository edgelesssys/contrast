module github.com/edgelesssys/contrast/imagestore

go 1.25.0

replace github.com/edgelesssys/contrast => ../

require (
	github.com/containerd/ttrpc v1.2.7
	github.com/edgelesssys/contrast v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.1
	golang.org/x/sync v0.17.0
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250826171959-ef028d996bc1 // indirect
)
