module github.com/edgelesssys/contrast/service-mesh

go 1.23.0

require (
	github.com/cncf/xds/go v0.0.0-20240723142845-024c85f92f20
	github.com/coreos/go-iptables v0.8.0
	github.com/envoyproxy/go-control-plane/envoy v1.32.2
	google.golang.org/protobuf v1.36.3
)

require (
	cel.dev/expr v0.16.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.1.0 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
)

// A refactoring of the go-control-plane library let's Go detect ambiguous imports:
//
// config.go:13:2: ambiguous import: found package github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3 in multiple modules:
//   github.com/envoyproxy/go-control-plane v0.13.1 (go/pkg/mod/github.com/envoyproxy/go-control-plane@v0.13.1/envoy/config/bootstrap/v3)
//   github.com/envoyproxy/go-control-plane/envoy v1.32.2 (go/pkg/mod/github.com/envoyproxy/go-control-plane/envoy@v1.32.2/config/bootstrap/v3)
//
// We point to the newer v0.13.2 version of go-control-plane when resolving the import path, even
// though the module isn't used after the split.
replace github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.13.2
