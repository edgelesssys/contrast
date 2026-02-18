# RFC 013: Configuration File

## Background

Contrast is currently configured exclusively via a CLI with a large number of flags and arguments across multiple subcommands.
While this provides flexibility, it leads to several practical issues:
- Users who want to make changes to a deployment must reconstruct long command invocations.
- In team settings, these invocations must be shared out-of-band.
- Configuration intent isn't easily reviewable or version-controllable.

As a result, users are effectively forced to build their own configuration abstraction on top of the CLI.
This RFC proposes introducing a first-class configuration file to address this, while preserving full backward compatibility with existing CLI usage.

## Requirements

1. Existing CLI invocations must continue to work without modification. More specifically:
    1. Command-line flags and arguments must always take precedence over configuration file values.
    2. Users must not be required to use a configuration file.
2. The configuration format should easily be extensible for future use cases.
3. The introduction of a configuration file must not significantly increase maintenance burden.

## Design

### Configuration format

We introduce a TOML-based configuration file.
TOML is chosen for its readability and explicit structure.
Additionally, parts of the codebase already interact with TOML files, allowing us to avoid additional dependencies.

The configuration file makes use of TOML tables for structure.
At its most basic, the file only contains a `cli` table, with its keys corresponding to the existing CLI arguments:

```toml
[cli]
# path to the policy (.rego) file
policy = "./path/to/policy.rego"

# path to the settings (.json) file
settings = "./path/to/settings.json"

# path to the cache (.json) file containing the image layers
genpolicy-cache-path = "./path/to/layers-cache.json"

# path to the manifest (.json) file
manifest = "./path/to/manifest.json"

# set the default reference values used for attestation
reference-values = "Metal-QEMU-SNP"

# ...
```

All configuration relevant to the CLI is nested under this section.
Additional sections may be introduced as needed, but a single configuration file is used for the entire project.

All fields are optional; if users specify a required field neither in the configuration file nor in their CLI arguments,
the existing argument validation logic will inform the user in the same way it currently does.

### Mapping CLI flags to configuration

The configuration file mirrors the existing CLI surface:
all flags, arguments, and options have corresponding fields in the configuration.

Command-line arguments always take precedence over values loaded from the configuration file.
This ensures that:
- Users who don't use a configuration file experience no behavior changes.
- Existing tooling and scripts continue to work unchanged.
- Users may selectively override configuration values for one-off invocations.

### Duplicate fields

Contrast subcommands do share a subset of their arguments.
For example, `generate`, `set`, `verify` and `recover` all require setting the `manifest` argument.

We could adapt the structure of the configuration file to match this, for example:

```toml
[cli.generate]
# path to the manifest (.json) file
manifest = "./path/to/manifest.json"
# ...

[cli.set]
# path to the manifest (.json) file
manifest = "./path/to/manifest.json"
# ...

[cli.verify]
# path to the manifest (.json) file
manifest = "./path/to/manifest.json"
# ...

[cli.recover]
# path to the manifest (.json) file
manifest = "./path/to/manifest.json"
# ...
```

However, the need for this seems questionable: for all options shared between the commands, values aren't expected to differ.
If a user does require overriding such a shared argument in a single invocation, they can always override the configuration file value in the CLI command.
The procedure to obtain the relevant configuration options for a subcommand from the configuration file are shown below.

### Internal representation

Internally, the configuration is represented as a single, unified Go struct.
This struct contains all configuration fields relevant to the application, and serves as the single source of truth for configuration options.
More on this below.

```go
type Config struct {
	CLI struct {
		// shared
		LogLevel     string `toml:"log-level" comment:"set logging level (debug, info, warn, error, or a number)"`
		WorkspaceDir string `toml:"workspace-dir" comment:"directory to write files to, if not set explicitly to another location"`

		ManifestPath         string `toml:"manifest" comment:"path to manifest (.json) file"`
		Coordinator          string `toml:"coordinator" comment:"endpoint the coordinator can be reached at"`
		PolicyPath           string `toml:"policy-path" comment:"path to policy (.rego) file"`
		WorkloadOwnerKeyPath string `toml:"workload-owner-key" comment:"path to workload owner key (.pem) file"`
		// ...

		// generate-specific
		SettingsPath            string   `toml:"settings" comment:"path to settings (.json) file"`
		GenpolicyCachePath      string   `toml:"genpolicy-cache-path" comment:"path to cache for the cache (.json) file containing the image layers"`
		ReferenceValuesPlatform string   `toml:"reference-values" comment:"set the default reference values used for attestation"`
		WorkloadOwnerKeys       []string `toml:"add-workload-owner-keys" comment:"add a workload owner key from a PEM file to the manifest (set more than once to add multiple keys)"`
		// ...

		// set-specific
		// ...
	}
}
```

The struct makes use of the `toml` and `comment` annotations provided by [pelletier/go-toml](https://pkg.go.dev/github.com/pelletier/go-toml/v2#example-Marshal-Commented).
A function `Default` is added to create a instance of the struct with default values set mirroring their current defaults in the `cobra.Command` subcommands:

```go
func Default() Config {
	return Config{
		CLI: {
			// shared
			LogLevel: "warn",

			ManifestPath:         "manifest.json",
			PolicyPath:           "rules.rego",
			WorkloadOwnerKeyPath: "workload-owner.pem",
			// ...

			// generate-specific
			SettingsPath:       "settings.json",
			GenpolicyCachePath: "layers-cache.json",
			WorkloadOwnerKeys:  []string{"workload-owner.pem"},
			// ...

			// set-specific
			// ...
		},
	}
}
```

The `Default` function has two main purposes.
First, by creating a `Config` object through it, then unmarshaling a TOML configuration into the resulting struct, analogous behavior to the current one (that is, some missing flags fall back to defaults) is achieved.
Secondly, we can trivially marshal the default `Config` object to obtain a commented, default-configured TOML file which users can use as a starting point for their configuration.

### `cobra.Command` derivations

The `Config` struct as depicted above uses the `toml` and `comment` annotations.
An optional `short:"<char>"` annotation is added, that is,

```go
type Config struct {
	CLI struct {
		// ...
		PolicyPath string `toml:"policy-path" short:"p" comment:"path to policy (.rego) file"`
		// ...
	}
}
```

and so forth.
Together with the `Default` function, this provides all necessary information to derive the Cobra (sub-)commands.
The functions currently used to create the `cobra.Command` structs for (sub-)commands get simplified as shown below for `NewGenerateCommand`.

```go
func NewGenerateCmd(cfg Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate [flags] paths...",
		Short: "generate policies and inject into Kubernetes resources",
		Long:  `Generate policies and inject into the given Kubernetes resources. [...]`,
		RunE:  withTelemetry(runGenerate),
	}
	// ...

	AddArgs(cfg, cmd, []string{
		"policy",
		"settings",
		"genpolicy-cache-path",
		"manifest",

		"workload-owner-key",
		"disable-updates",
		// ...
	})

	cmd.MarkFlagsMutuallyExclusive("add-workload-owner-key", "disable-updates")
	return cmd
}
```

Here, `AddArgs` takes the provided (default) `Config` and the `cobra.Command`, as well as a slice of argument names matching the ones used in the `toml` annotations.
It then adds a `cobra.Command` argument for each one of those names, using the metadata from the `toml` annotations for short names and help texts.

Validation of the provided arguments continues to work just as it currently does (for example `parseGenerateFlags`), with the only difference being that this also runs on the new `Config` struct.

### Configuration loading and precedence

The `contrast` root command receives a new optional, persistent flag `--config`.
Upon invocation of any CLI (sub-)command, the following steps are performed:
- In the `OnInitialize` function of the root command:
    - Create a `Config` object via `Default()`
    - If the `--config` flag was set, load the specified configuration file.
    - If the file is specified, but missing or can't be parsed, exit with error.
    - Unmarshal the config file into the default configuration.
    - Pass the config struct to the subcommands.
- In each subcommand:
    - For all flags set in the CLI invocation, override the corresponding `Config` struct field.
    - Apply the current validation logic.

### Parsing behavior

When parsing the configuration file, unknown fields intentionally result in an error to prevent typos or use of deprecated fields.
Missing fields don't result in errors, that is, all fields are optional in the configuration file.

### Versioning

A version field may be included in the configuration file to allow explicit handling of breaking changes:

```toml
version = 1.17
```

This would allow us to perform compatibility checks when loading older configurations.
However, versioning is considered optional at this stage and may be introduced later if required.

### Backward and Forward Compatibility

Backward compatibility is ensured by:
- Making the configuration file entirely optional.
- Giving precedence to CLI flags over configuration values.
- Preserving the existing CLI interface.

## Further applications

In addition to using the configuration file for simplifying CLI uses, other applications could also be considered.
The structure of the configuration file and the `Config` struct, that is, a nested `CLI` struct or table inside it,
serves the purposes of compartmentalizing these different applications.

One such additional application is sketched out below.

### Reference values

Currently, a `--reference-values` argument needs to be passed to the CLI in (most) invocations of `generate`.
Afterward, users need to manually fill in the actual reference values for the specified platforms in the manifest.

Allowing the users to instead set these values directly in the configuration file provides a dedicated place to store these values across manifest lifetimes,
and to populate the manifest directly from this file, without additional user involvement.
```toml
[[reference_values]]
platform = "Metal-QEMU-SNP"
patch = '''
[
    ...
]
'''
# ...
```

The actual values here are JSON-patches, in keeping with how we handle our own reference value patches.

Again, these sections should be completely optional.
Either passing `--reference-values` to the CLI *or* setting `cli.reference_values` in the CLI section of the configuration file should preclude the use of these values.

## Alternatives considered

### Using Viper

The [Viper](https://github.com/spf13/viper) library was considered due to its easy integration with Cobra and the built-in support for config files.
However, Viper is geared more towards interacting with a configuration file from within an application, that is, loading, editing and saving a config file.
Working with a single configuration struct and deriving the subcommands from this configuration also doesn't appear simpler than implementing the above suggestions manually.
