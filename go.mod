module github.com/edgelesssys/contrast

go 1.24.0

toolchain go1.24.2

// The upstream package has some stepping issues with Genoa:
// https://github.com/google/go-sev-guest/issues/115
// https://github.com/google/go-sev-guest/issues/103
// Includes cherry-pick of unmerged PR to fix platform info validation:
// https://github.com/google/go-sev-guest/pull/161
// Includes cherry-pick of unmerged PR for Siena support:
// https://github.com/google/go-sev-guest/pull/167
replace github.com/google/go-sev-guest => github.com/edgelesssys/go-sev-guest v0.0.0-20250522132418-02d4f5059bf3

require (
	filippo.io/keygen v0.0.0-20240718133620-7f162efbbd87
	github.com/coreos/go-iptables v0.8.0
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/elazarl/goproxy v1.7.2
	github.com/google/go-github/v72 v72.0.0
	github.com/google/go-sev-guest v0.13.0
	github.com/google/go-tdx-guest v0.3.2-0.20250505161510-9efd53b4a100
	github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus v1.0.1
	github.com/katexochen/sync v0.0.0-20240617152407-6a8003240db0
	github.com/klauspost/cpuid/v2 v2.2.10
	github.com/pelletier/go-toml/v2 v2.2.4
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/common v0.64.0
	github.com/spf13/afero v1.14.0
	github.com/spf13/cobra v1.9.1
	github.com/stretchr/testify v1.10.0
	go.uber.org/goleak v1.3.0
	golang.org/x/crypto v0.38.0
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6
	golang.org/x/sync v0.14.0
	golang.org/x/sys v0.33.0
	golang.org/x/term v0.32.0
	google.golang.org/grpc v1.72.1
	google.golang.org/protobuf v1.36.6
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.33.1
	k8s.io/apimachinery v0.33.1
	k8s.io/client-go v0.33.1
	k8s.io/utils v0.0.0-20250502105355-0f33e8f1c979
	sigs.k8s.io/yaml v1.4.0
)

require (
	filippo.io/bigmod v0.0.3 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/godbus/dbus/v5 v5.0.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-configfs-tsm v0.3.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/logger v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/time v0.9.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250318190949-c8a335a9a2ff // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.6.0 // indirect
)
