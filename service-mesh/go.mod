module github.com/edgelesssys/contrast/service-mesh

go 1.26.0

replace github.com/edgelesssys/contrast => ../

require (
	github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2
	github.com/coreos/go-iptables v0.8.0
	github.com/edgelesssys/contrast v0.0.0-00010101000000-000000000000
	github.com/envoyproxy/go-control-plane/envoy v1.40.0
	github.com/stretchr/testify v1.12.1
	go.uber.org/goleak v1.3.0
	google.golang.org/protobuf v1.36.12
)

require (
	cel.dev/expr v0.25.2 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/google/logger v1.1.2 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20250313105119-ba97887b0a25 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.49.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260720211330-0afa2a65878a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260720211330-0afa2a65878a // indirect
)
