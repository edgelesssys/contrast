module github.com/edgelesssys/contrast/securemount

go 1.24.0

replace github.com/edgelesssys/contrast => ../

require (
	github.com/containerd/ttrpc v1.2.7
	github.com/edgelesssys/contrast v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.9.1
	golang.org/x/sync v0.16.0
	google.golang.org/protobuf v1.36.7
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
	golang.org/x/sys v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/grpc v1.74.2 // indirect
)
