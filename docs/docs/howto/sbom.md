# Software bill of materials

This page lists the SBOMs published with each Contrast release and shows how to scan them.

:::warning

A scan of these SBOMs reports vulnerabilities that Contrast already tracks and has assessed as not exploitable in our configuration.
The raw finding count from any scanner is therefore higher than the number of issues that affect you, and a sizeable share of those findings carries a high or critical rating.
Contrast records each assessment in [`osv-scanner.toml`](https://github.com/edgelesssys/contrast/blob/main/osv-scanner.toml) and [`vulnix-whitelist.toml`](https://github.com/edgelesssys/contrast/blob/main/vulnix-whitelist.toml), and the scans we run in CI apply them.
Read a scan of your own against those files before acting on it.

:::

## Applicability

For vulnerability management, license compliance, and regulatory requirements such as the EU Cyber Resilience Act and [BSI TR-03183-2](https://www.bsi.bund.de/SharedDocs/Downloads/EN/BSI/Publications/TechGuidelines/TR03183/BSI-TR-03183-2_v2_1_0.pdf?__blob=publicationFile).

## Prerequisites

A CycloneDX-aware scanner, for example [Grype](https://github.com/anchore/grype), [Trivy](https://github.com/aquasecurity/trivy), or [Dependency-Track](https://dependencytrack.org/).

## How-to

### Published SBOMs

Every release publishes these assets in [CycloneDX](https://cyclonedx.org/) JSON format:

| Asset | Covers |
| --- | --- |
| `contrast-cli-x86_64-linux.cdx.json` | The Linux CLI release binary, including the genpolicy it embeds |
| `contrast-cli-aarch64-darwin.cdx.json` | The same for the macOS CLI release binary |
| `contrast-runtime-confidential.cdx.json` | Everything that runs inside a confidential VM: Coordinator, initializer, service mesh, image puller and store, initdata processor, guest kernel and image, and the SNP and TDX guest firmware |
| `contrast-runtime-non-confidential.cdx.json` | The parts that run on the host: node installer and the Kata runtime |

The CLI SBOM is per platform because each platform ships a different binary.
Pick the one matching the binary you downloaded.

### Scan an SBOM

Download the SBOM and scan it without unpacking or rebuilding anything:

```bash
version=$(gh release view --repo edgelesssys/contrast --json tagName -q .tagName)
gh release download "$version" --repo edgelesssys/contrast --pattern 'contrast-runtime-confidential.cdx.json'
grype sbom:contrast-runtime-confidential.cdx.json
```

Contrast runs the same kind of scan in CI, using `osv-scanner` and `vulnix`:

```bash
nix run .#base.scripts.vuln-scan -- contrast-runtime-confidential.cdx.json
```

### Match a binary to its SBOM

The CLI SBOM records the digest of the published binary, so a tool can pair the two without a rebuild:

```bash
sha256sum contrast
jq -r '.metadata.component.hashes[] | select(.alg == "SHA-256") | .content' contrast-cli-x86_64-linux.cdx.json
```

### Accuracy and integrity

The SBOMs are generated from the same Nix build that produces the release artifacts, so they describe what's actually shipped rather than what a scanner infers from a finished binary.

They're regular release assets, so `gh release verify-asset` checks them like any other file:

```bash
gh release verify-asset "$version" contrast-runtime-confidential.cdx.json --repo edgelesssys/contrast
```
