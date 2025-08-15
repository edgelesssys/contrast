module github.com/edgelesssys/contrast/imagepuller-benchmark

go 1.24.0

replace github.com/edgelesssys/contrast/imagepuller => ../../imagepuller

require (
	github.com/edgelesssys/contrast/imagepuller v0.0.0-00010101000000-000000000000
	github.com/shirou/gopsutil/v4 v4.25.7
	github.com/spf13/cobra v1.9.1
	golang.org/x/sys v0.35.0
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/ttrpc v1.2.7 // indirect
	github.com/ebitengine/purego v0.8.4 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/grpc v1.69.0 // indirect
	google.golang.org/protobuf v1.36.7 // indirect
)
