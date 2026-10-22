# Software bill of materials

This guide shows how to use the software bills of materials (SBOMs) published with each Contrast release, for independent vulnerability scanning and for licensing and regulatory compliance.

:::warning

A scan of these SBOMs reports vulnerabilities that Contrast already tracks and has assessed as not exploitable in our configuration.
The raw findings count from any scanner is therefore higher than the number of issues actually affecting Contrast.
See [`vuln-ignores.toml`](https://github.com/edgelesssys/contrast/blob/main/vuln-ignores.toml) for details regarding the inapplicability of vulnerabilities to Contrast.
Please check this file before deciding if local scan results require action.

:::

## Applicability

For vulnerability management, license compliance, and regulatory requirements such as the EU Cyber Resilience Act and [BSI TR-03183-2](https://www.bsi.bund.de/SharedDocs/Downloads/EN/BSI/Publications/TechGuidelines/TR03183/BSI-TR-03183-2_v2_1_0.pdf?__blob=publicationFile).
Contrast aims for the highest standards the BSI sets for SBOM content and quality.

## Prerequisites

1. The [GitHub CLI](https://cli.github.com/) and [jq](https://jqlang.org/), to download and read the assets
2. For vulnerabilities: [Grype](https://github.com/anchore/grype) or [Trivy](https://github.com/aquasecurity/trivy) for the Go and Rust components, [vulnix](https://github.com/nix-community/vulnix) for the Nix components
3. For licenses: [Trivy](https://github.com/aquasecurity/trivy)
4. For BSI TR-03183-2 conformance: [sbomqs](https://github.com/interlynk-io/sbomqs)

## How-to

### Published SBOMs

Every release publishes these assets in the BSI-recommended [CycloneDX](https://cyclonedx.org/) JSON format:

| Asset                                        | Covers                                                                                                                                                                                        |
| -------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `contrast-cli-x86_64-linux.cdx.json`         | The Linux CLI release binary                                                                                                                                                                  |
| `contrast-cli-aarch64-darwin.cdx.json`       | The macOS CLI release binary                                                                                                                                                                  |
| `contrast-runtime-confidential.cdx.json`     | Everything that runs inside a confidential VM: Coordinator, initializer, service mesh, image puller and store, initdata processor, guest kernel and image, and the SNP and TDX guest firmware |
| `contrast-runtime-non-confidential.cdx.json` | The parts that run on the host: node installer and Kata runtimes                                                                                                                              |

### Download an SBOM

```bash
version=$(gh release view --repo edgelesssys/contrast --json tagName -q .tagName)
gh release download "$version" --repo edgelesssys/contrast --pattern 'contrast-runtime-confidential.cdx.json'
```

### Scan for vulnerabilities

Grype and Trivy resolve the Go and Rust components:

```bash
grype sbom:contrast-runtime-confidential.cdx.json
```

Public vulnerability databases don't index the `pkg:nix` purls of the Nix components, so those go to vulnix, which reads them as a Nix package list:

```bash
jq '[.components[] | select((.purl // "") | startswith("pkg:nix"))
     | {key: .["bom-ref"], value: {name: "\(.name)-\(.version)", patches: []}}] | from_entries' \
  contrast-runtime-confidential.cdx.json >nix-packages.json
vulnix --from-file nix-packages.json
```

Contrast runs both kinds of scan in CI on every release and nightly, so that we can respond to emerging advisories.
With Nix at hand, you can repeat that scan with our assessments applied, against an SBOM built from the release tag:

```bash
nix run "github:edgelesssys/contrast/${version}#base.scripts.vuln-scan" -- \
  "$(nix build --no-link --print-out-paths "github:edgelesssys/contrast/${version}#base.sbom.runtimeConfidential")"
```

It downloads the components themselves, and covers the Go, Rust and Nix findings in one report.

### Check licenses

```bash
trivy sbom --scanners license contrast-runtime-confidential.cdx.json
```

### Check BSI conformance

```bash
sbomqs score --profile bsi-v2.1 --detailed contrast-runtime-confidential.cdx.json
```

### Match a binary to its SBOM

The CLI SBOMs record the digests of the published binaries, so a tool can pair the two automatically:

```bash
sha256sum contrast-x86_64-linux
jq -r '.metadata.component.hashes[] | select(.alg == "SHA-256") | .content' contrast-cli-x86_64-linux.cdx.json
```

The digest identifies which binary an SBOM describes, so tooling can pair the two without going by file name.
Trust in the content comes from the release attestation below, or from rebuilding both from the tag.

### Integrity

The SBOMs and the binaries are regular release assets, so `gh release verify-asset` checks each file you downloaded, including the pair whose digests you compared above:

```bash
gh release verify-asset "$version" contrast-runtime-confidential.cdx.json --repo edgelesssys/contrast
gh release verify-asset "$version" contrast-cli-x86_64-linux.cdx.json --repo edgelesssys/contrast
gh release verify-asset "$version" contrast-x86_64-linux --repo edgelesssys/contrast
```
