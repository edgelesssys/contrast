# Software bill of materials

This guide shows how to use the software bills of materials (SBOMs) published with each Contrast release, for independent vulnerability scanning and for licensing and regulatory compliance.

:::warning

A scan of these SBOMs reports vulnerabilities that Contrast already tracks and has assessed as not exploitable in our configuration.
The raw finding count from any scanner is therefore higher than the number of issues that affect you, and a sizeable share of those findings carries a high or critical rating.
Contrast records each assessment in [`osv-scanner.toml`](https://github.com/edgelesssys/contrast/blob/main/osv-scanner.toml) and [`vulnix-whitelist.toml`](https://github.com/edgelesssys/contrast/blob/main/vulnix-whitelist.toml), and the scans we run in CI apply them.
Read a scan of your own against those files before acting on it.

:::

## Applicability

For vulnerability management, license compliance, and regulatory requirements such as the EU Cyber Resilience Act and [BSI TR-03183-2](https://www.bsi.bund.de/SharedDocs/Downloads/EN/BSI/Publications/TechGuidelines/TR03183/BSI-TR-03183-2_v2_1_0.pdf?__blob=publicationFile).
Contrast aims to meet the standards the BSI sets for SBOM content and quality.

## Prerequisites

The [GitHub CLI](https://cli.github.com/) to download the assets, plus whichever tools match what you want to check:

| Tool | Checks |
| --- | --- |
| [Grype](https://github.com/anchore/grype) or [Trivy](https://github.com/aquasecurity/trivy) | Vulnerabilities in the Go and Rust components |
| [vulnix](https://github.com/nix-community/vulnix) | Vulnerabilities in the Nix components, which the other scanners skip |
| [Trivy](https://github.com/aquasecurity/trivy) | Licenses |
| [sbomqs](https://github.com/interlynk-io/sbomqs) | BSI TR-03183-2 conformance |

## How-to

### Published SBOMs

Every release publishes these assets in the BSI-recommended [CycloneDX](https://cyclonedx.org/) JSON format:

| Asset | Covers |
| --- | --- |
| `contrast-cli-x86_64-linux.cdx.json` | The Linux CLI release binary |
| `contrast-cli-aarch64-darwin.cdx.json` | The macOS CLI release binary |
| `contrast-runtime-confidential.cdx.json` | Everything that runs inside a confidential VM: Coordinator, initializer, service mesh, image puller and store, initdata processor, guest kernel and image, and the SNP and TDX guest firmware |
| `contrast-runtime-non-confidential.cdx.json` | The parts that run on the host: node installer and Kata runtimes |

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

The Nix components carry a `pkg:nix` purl, which public vulnerability databases don't index, so those need vulnix:

```bash
vulnix --closure contrast-runtime-confidential.cdx.json
```

Contrast runs both kinds of scan in CI on every release and nightly, so that we can respond to emerging advisories.

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

Together with a reproducible build, that digest lets you verify the whole chain from our source to the binary you run.

### Integrity

The SBOMs are regular release assets, so `gh release verify-asset` checks them like any other file:

```bash
gh release verify-asset "$version" contrast-runtime-confidential.cdx.json --repo edgelesssys/contrast
```
