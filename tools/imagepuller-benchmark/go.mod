module github.com/edgelesssys/contrast/imagepuller-benchmark

go 1.25.5

replace github.com/edgelesssys/contrast/imagepuller => ../../imagepuller

require (
	github.com/edgelesssys/contrast/imagepuller v0.0.0-00010101000000-000000000000
	github.com/shirou/gopsutil/v4 v4.25.11
	github.com/spf13/cobra v1.10.1
	golang.org/x/sys v0.38.0
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/ttrpc v1.2.7 // indirect
	github.com/ebitengine/purego v0.9.1 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/sirupsen/logrus v1.9.4-0.20230606125235-dd1b4c2e81af // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250826171959-ef028d996bc1 // indirect
	google.golang.org/grpc v1.75.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
