# Contrast SDK

**Caution:** This SDK is still under active development and not fit for external use yet.
Please expect breaking changes with new minor versions.

The SDK allows writing programs that interact with a Contrast deployment like the CLI does, without relying on the CLI.

# Building

If you decide to use the unstable API and accept the risk of breakage, you need to set the Go build tag `contrast_unstable_api`.

# Versioning

The SDK is a Go module of its own, `github.com/edgelesssys/contrast/sdk`.
It's released together with Contrast and shares its minor and patch version, with a major version of `v0`:
Contrast `v1.25.0` comes with `sdk/v0.25.0` and `apitypes/v0.25.0`, all tagged on the same commit.
The Contrast release requires the SDK version it was tagged together with.

The Contrast module replaces some of its dependencies with forks.
Go ignores `replace` directives of dependencies, so you need to copy the `replace` directives for `github.com/google/go-sev-guest` and `github.com/google/go-tdx-guest` from the `go.mod` of the SDK into your own.
