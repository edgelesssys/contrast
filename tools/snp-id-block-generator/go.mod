module github.com/edgelesssys/contrast/snp-id-block-generator

go 1.25.0

replace github.com/edgelesssys/contrast => ../..

replace github.com/edgelesssys/contrast/tools/igvm => ../igvm

require (
	github.com/edgelesssys/contrast v0.0.0
	github.com/edgelesssys/contrast/tools/igvm v0.0.0
	github.com/google/go-sev-guest v0.14.1
	github.com/spf13/afero v1.15.0
	github.com/spf13/cobra v1.10.1
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/logger v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
