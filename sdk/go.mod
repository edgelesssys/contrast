module github.com/edgelesssys/contrast/sdk

go 1.26.0

replace github.com/edgelesssys/contrast => ../

replace github.com/edgelesssys/contrast/apitypes => ../apitypes

// Upstream is poorly maintained, use edgelesssys fork instead.
replace github.com/google/go-tdx-guest => github.com/edgelesssys/go-tdx-guest v0.0.0-20261013140315-ce43fe5589cd

replace github.com/google/go-sev-guest => github.com/edgelesssys/go-sev-guest v0.0.0-20261013132829-7a9ace10598e

require (
	github.com/edgelesssys/contrast v1.26.0
	github.com/edgelesssys/contrast/apitypes v0.26.0
	github.com/spf13/afero v1.15.0
	github.com/stretchr/testify v1.12.1
)

require (
	filippo.io/nistec v0.0.4 // indirect
	github.com/edgelesssys/go-sev-guest v0.0.0-20261027073724-e4b917838318 // indirect
	github.com/edgelesssys/go-tdx-guest v0.0.0-20261027073737-15e4e62a1ff7 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/google/logger v1.1.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mdlayher/socket v0.6.1 // indirect
	github.com/mdlayher/vsock v1.3.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.58.0 // indirect
	golang.org/x/exp v0.0.0-20261014143641-f0251eb4aa3e // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/grpc v1.84.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	k8s.io/utils v0.0.0-20261023123148-ca04ba58ecab // indirect
)
