module github.com/edgelesssys/contrast/service-mesh

go 1.25.0

replace github.com/edgelesssys/contrast => ..

require (
	github.com/cncf/xds/go v0.0.0-20251210132809-ee656c7534f5
	github.com/coreos/go-iptables v0.8.0
	github.com/edgelesssys/contrast v0.0.0-00010101000000-000000000000
	github.com/envoyproxy/go-control-plane/envoy v1.36.0
	github.com/stretchr/testify v1.11.1
	go.uber.org/goleak v1.3.0
	google.golang.org/protobuf v1.36.11
)

require (
	cel.dev/expr v0.24.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/google/logger v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20250313105119-ba97887b0a25 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/sys v0.40.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250826171959-ef028d996bc1 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
