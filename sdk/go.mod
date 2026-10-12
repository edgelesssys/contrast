module github.com/edgelesssys/contrast/sdk

go 1.26.0

replace github.com/edgelesssys/contrast => ../

replace github.com/edgelesssys/contrast/apitypes => ../apitypes

// Upstream is poorly maintained, use edgelesssys fork instead.
replace github.com/google/go-tdx-guest => github.com/edgelesssys/go-tdx-guest v0.0.0-20261013140315-ce43fe5589cd

replace github.com/google/go-sev-guest => github.com/edgelesssys/go-sev-guest v0.0.0-20261013132829-7a9ace10598e

require (
	github.com/edgelesssys/contrast v1.25.0
	github.com/edgelesssys/contrast/apitypes v0.25.0
	github.com/spf13/afero v1.15.0
	github.com/stretchr/testify v1.11.1
)

require (
	filippo.io/nistec v0.0.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/google/go-sev-guest v0.14.2-0.20251119154202-af1c107a648f // indirect
	github.com/google/go-tdx-guest v0.3.2-0.20260104162950-32866d7a678f // indirect
	github.com/google/logger v1.1.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mdlayher/socket v0.6.1 // indirect
	github.com/mdlayher/vsock v1.2.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/exp v0.0.0-20260218203240-3dfff04db8fa // indirect
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260610212136-7ab31c22f7ad // indirect
	google.golang.org/grpc v1.83.2 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/utils v0.0.0-20260210185600-b8788abfbbc2 // indirect
)
