module github.com/edgelesssys/contrast/initdata-processor

go 1.25.0

replace (
	github.com/edgelesssys/contrast => ..
	github.com/google/go-sev-guest => github.com/edgelesssys/go-sev-guest v0.0.0-20250811150530-d85b756e97f2
)

require (
	github.com/edgelesssys/contrast v0.0.0-00010101000000-000000000000
	github.com/google/go-sev-guest v0.14.1
	github.com/google/go-tdx-guest v0.3.2-0.20250814004405-ffb0869e6f4d
	github.com/stretchr/testify v1.11.1
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/go-configfs-tsm v0.3.3 // indirect
	github.com/google/logger v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
