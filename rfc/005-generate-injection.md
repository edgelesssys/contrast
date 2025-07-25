# RFC 005: Inject Initializer and Service Mesh in Contrast CLI

## Problem

Currently, the `generate` command in the Contrast CLI doesn't include the
functionality to automatically inject the Contrast initializer and service mesh
components into the user's Kubernetes YAML resources. This poses several
challenges for users: they need to manually copy and paste initializer and
service mesh configurations from the documentation into their YAML files, which
is time-consuming, error-prone, and disrupts the workflow. They have to keep
checking that they're using versions of the initializer and service mesh images
that are compatible with their version of Contrast, and update them manually
when these components change across versions. This leads to maintenance overhead
and inconsistencies. Automating the injection of the initializer and service
mesh components provides a simpler, more streamlined experience.

## Proposed Solution

### Initializer Injection

By default, the `generate` command will iterate through the pod specs in the
provided YAML files and for each pod spec that has the `contrast-cc` runtime
class set, an init container with the Contrast initializer image will be added
before any other existing containers in the pod spec. We will then provide two
mechanisms for users to opt-out of the initializer injection.

- **Per object (annotation)**: Allow users to opt-out on a per-object basis by
  adding an annotation (like `contrast.edgeless.systems/skip-initializer`) to
  the Kubernetes object.
- **Per `contrast generate` (command-line flag)**: Users can opt-out of
  initializer injection for an entire `generate` invocation by providing the
  `--skip-initializer` flag. If this flag is set, the initializer won't be
  injected into any pod spec whatsoever.

#### Ensuring Reliable Initializer Image Embedding

We need to ensure that the Contrast always uses the correct version of the
initializer image that's compatible with the current version.

During the release build process, the `packages/by-name/cli-release/package.nix`
file will be updated to embed the published initializer image reference into the
Contrast binary. For development builds, we can use an `--image-replacements`
flag to specify a file containing image replacements. This flag will be marked
as hidden using `pflag.MarkHidden()`, as it's not intended for regular user
consumption.

Here is a proposed way to do this that optimizes for the common case where most
workloads will use the Contrast initializer, but still provides flexibility for
users to opt-out at different levels of granularity:

1. Add `--image-replacements` string and `--skip-initializer` Boolean flags to
   the `generateFlags` struct and `parseGenerateFlags` function in
   `cli/cmd/generate.go`. In `runGenerate`, after updating the policies, check
   the value of the `--skip-initializer` flag.

   - If the flag is set to true, skip the initializer injection process entirely
     and proceed with writing the updated YAML files.
   - If the flag is set to false or not provided, proceed with the per-object
     annotation checks.

2. Iterate over the un-marshalled K8s resources:

   - If the pod has the `contrast.edgeless.systems/skip-initializer` annotation,
     skip injection for that specific pod spec.
   - If the pod spec doesn't have the skip annotation, proceed with injection by
     calling an `injectInitializer` function (to be written, which can use
     `kuberesource.Initializer` and `kuberesource.PatchImages`) with the pod
     spec and the selected initializer image reference, or a default one if the
     `--image-replacements` flag isn't set. If any existing `initContainers`
     have the same name as the Contrast initializer, the existing init container
     will be updated with the injected Contrast initializer as described in the
     edge cases.

3. Re-encode and write the updated YAML.

#### Edge cases

To make `contrast generate` idempotent and handle potential edge cases, we will
implement the following behavior:

1. _Container name conflict:_
   - Use a highly unique name for the initializer container, like
     `contrast-initializer.` This virtually eliminates the chances of name
     conflicts with user containers.
   - If an `initContainer` with the same unique name already exists, overwrite
     it with the current version of the Contrast initializer.
   - If no matching `initContainer` is found, inject a new one using the current
     version of the Contrast initializer and the unique name.
   - If other `initContainers` exist, insert the Contrast initializer as the
     first one in the list.

2. _Volume Name Conflict:_
   - Use a highly unique name for the volume used by the Contrast initializer,
     such as `contrast-tls-certs.` This reduces the chances of name conflicts
     with existing volumes.
   - If a pod spec already contains a volume with the same unique name, no
     action will be taken, and the existing volume will be reused.

### Service Mesh Injection

The service mesh injection will follow a similar approach to the initializer
injection outlined, apart from being opt-in instead of enabled by default.
During the generation process, all containers with the `contrast-cc` runtime
class and a specified service mesh proxy configuration will have a service mesh
added as a sidecar init container.

The configuration for the Envoy proxy is handled via Kubernetes object
annotations. The annotations `contrast.edgeless.systems/servicemesh-ingress`,
`contrast.edgeless.systems/servicemesh-egress` and
`contrast.edgeless.systems/servicemesh-admin-interface-port` will be written
into the environment variables `EDG_INGRESS_PROXY_CONFIG`,
`EDG_EGRESS_PROXY_CONFIG` and `EDG_ADMIN_PORT` of the injected service mesh
container.

As long as one of the corresponding annotations is present on an object, a
service mesh sidecar container will be injected. To configure a service mesh
with default configuration, the annotation can be left empty.

#### Edge Cases

If a workload already contains a service mesh as an init container, it will be
replaced by the injection mechanism. Changing the environment variables of the
service mesh init container directly will therefore have no effect because the
entire service mesh container will be replaced on `contrast generate` using the
proxy configuration defined in the annotations.

### UX Considerations

- Document the injection behavior in the command's help text and docs,
  particularly how the command-line flag and annotations interact and which
  takes precedence.
- Provide logging and error messages in case of failures during the injection
  process.
- Add an `--output` flag to allow Unix pipeline capabilities. Standard behavior
  for such flags seems to be:
  - If not provided, `generate` command will as before write the updated YAML
    files back to disk, overwriting the original file.
  - If `--output` is set to `-`, `generate` will write the updated YAML to
    `stdout`.
  - If `--output` is set to a file path, write the updated YAML to the specified
    file.
