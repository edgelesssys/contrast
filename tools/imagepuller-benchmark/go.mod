module github.com/edgelesssys/contrast/imagepuller-benchmark

go 1.25.0

replace github.com/edgelesssys/contrast/imagepuller => ../../imagepuller

require (
	github.com/edgelesssys/contrast/imagepuller v0.0.0-00010101000000-000000000000
	github.com/shirou/gopsutil/v4 v4.26.1
	github.com/spf13/cobra v1.10.2
	golang.org/x/sys v0.41.0
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/ttrpc v1.2.7 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/grpc v1.79.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
