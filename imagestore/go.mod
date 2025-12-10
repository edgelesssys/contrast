module github.com/edgelesssys/contrast/imagestore

go 1.25.5

replace github.com/edgelesssys/contrast => ../

require (
	github.com/containerd/ttrpc v1.2.7
	github.com/edgelesssys/contrast v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.1
	golang.org/x/sync v0.18.0
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/sys v0.38.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8 // indirect
)
