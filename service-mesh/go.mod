module github.com/edgelesssys/contrast/service-mesh

go 1.24.0

replace github.com/edgelesssys/contrast => ..

require (
	github.com/cncf/xds/go v0.0.0-20250501225837-2ac532fd4443
	github.com/coreos/go-iptables v0.8.0
	github.com/edgelesssys/contrast v0.0.0-00010101000000-000000000000
	github.com/envoyproxy/go-control-plane/envoy v1.32.4
	github.com/stretchr/testify v1.10.0
	go.uber.org/goleak v1.3.0
	google.golang.org/protobuf v1.36.7
)

require (
	cel.dev/expr v0.19.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/google/logger v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/sys v0.35.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241202173237-19429a94021a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// A refactoring of the go-control-plane library let's Go detect ambiguous imports:
//
// config.go:13:2: ambiguous import: found package github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3 in multiple modules:
//   github.com/envoyproxy/go-control-plane v0.13.1 (go/pkg/mod/github.com/envoyproxy/go-control-plane@v0.13.1/envoy/config/bootstrap/v3)
//   github.com/envoyproxy/go-control-plane/envoy v1.32.2 (go/pkg/mod/github.com/envoyproxy/go-control-plane/envoy@v1.32.2/config/bootstrap/v3)
//
// We point to the newer v0.13.2 version of go-control-plane when resolving the import path, even
// though the module isn't used after the split.
replace github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.13.4
