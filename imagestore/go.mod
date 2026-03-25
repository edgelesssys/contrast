module github.com/edgelesssys/contrast/imagestore

go 1.25.6

replace github.com/edgelesssys/contrast => ../

require (
	github.com/containerd/ttrpc v1.2.8
	github.com/edgelesssys/contrast v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
	golang.org/x/sync v0.19.0
	google.golang.org/grpc v1.79.3
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/sys v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
