def rules:
  [ .[] as $p | $p.affected_by[] as $cve
    | { id: $cve,
        name: $cve,
        shortDescription: { text: ($cve + ": " + ($p.description[$cve] // "known vulnerability")) },
        helpUri: ("https://nvd.nist.gov/vuln/detail/" + $cve) } ]
  | unique_by(.id);

def results:
  [ .[] as $p | $p.affected_by[] as $cve
    | { ruleId: $cve,
        level: (if (($p.cvssv3_basescore[$cve] // 0) >= 7) then "error" else "warning" end),
        message: { text: ($p.name + " is affected by " + $cve
                    + (if $p.cvssv3_basescore[$cve]
                       then " (CVSS " + ($p.cvssv3_basescore[$cve] | tostring) + ")" else "" end)
                    + " [" + $p.derivation + "]") },
        locations: [ { physicalLocation: { artifactLocation: { uri: $sbom } } } ],
        partialFingerprints: { vulnixMatch: ($p.name + "/" + $cve) } } ];

{
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  version: "2.1.0",
  runs: [ {
    tool: { driver: {
      name: "vulnix",
      informationUri: "https://github.com/flyingcircusio/vulnix",
      rules: rules,
    } },
    results: results,
  } ],
}
