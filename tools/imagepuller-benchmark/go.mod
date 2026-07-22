module github.com/edgelesssys/contrast/imagepuller-benchmark

go 1.25.6

replace (
	github.com/edgelesssys/contrast => ../../
	github.com/edgelesssys/contrast/imagepuller => ../../imagepuller
)

require (
	github.com/edgelesssys/contrast/imagepuller v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
	golang.org/x/sys v0.46.0
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/ttrpc v1.2.8 // indirect
	github.com/edgelesssys/contrast v0.0.0-00010101000000-000000000000 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260610212136-7ab31c22f7ad // indirect
	google.golang.org/grpc v1.82.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
