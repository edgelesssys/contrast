module github.com/edgelesssys/contrast/initdata-processor

go 1.26.0

replace github.com/edgelesssys/contrast => ..

require (
	github.com/edgelesssys/contrast v0.0.0-00010101000000-000000000000
	github.com/edgelesssys/go-sev-guest v0.0.0-20261027073724-e4b917838318
	github.com/edgelesssys/go-tdx-guest v0.0.0-20261027073737-15e4e62a1ff7
	github.com/stretchr/testify v1.12.1
	golang.org/x/sys v0.49.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/google/go-configfs-tsm v0.3.3 // indirect
	github.com/google/logger v1.1.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.58.0 // indirect
)
