# Normalizes the JSON output of osv-scanner or vulnix into a common list of findings.
#
# Args:
#   $tool  "osv" or "vulnix", selecting the input format.
#
# Each finding is { tool, id, aliases, package, version, fixed, components, paths, title, score, link }.
# components and paths are filled in later by attribute.jq.

# Shortens NVD descriptions to their first sentence.
def first_sentence:
  gsub("\\s+"; " ") | ltrimstr(" ")
  | (capture("^(?<s>.*?[.!?])(?= [A-Z]|$)").s // .);

def parse_score: if . == null or . == "" then null else tonumber end;

# Comparable form of a version: "v1.2.3-rc.1" -> [1, 2, 3].
def version_parts: ltrimstr("v") | sub("[-+].*$"; "") | split(".") | map(tonumber? // 0);

# Advisories sometimes spell package names (mostly crates) with "_" where the SBOM has "-".
def normalized_package_name: ascii_downcase | gsub("_"; "-");

def osv_findings:
  [ .results[]?.packages[]? as $p
    | ($p.vulnerabilities | map({ (.id): . }) | add // {}) as $vulns
    | ($p.package.version | version_parts) as $current
    | $p.groups[]
    | .ids[0] as $id
    | { tool: "osv",
        id: $id,
        aliases: (.aliases - [ $id ]),
        package: $p.package.name,
        version: $p.package.version,
        # The lowest version that fixes any advisory of this group and is newer than the installed one.
        fixed: ([ .ids[] as $i
                  | $vulns[$i].affected[]?
                  | select(.package.ecosystem == $p.package.ecosystem
                           and (.package.name | normalized_package_name) == ($p.package.name | normalized_package_name))
                  | .ranges[]?.events[]?.fixed // empty ]
                | unique
                | map(select(version_parts > $current))
                | min_by(version_parts)),
        components: [],
        paths: [],
        title: ($vulns[$id].summary // ""),
        score: (.max_severity | parse_score),
        link: ("https://osv.dev/vulnerability/" + $id) } ];

def vulnix_findings:
  [ .[] as $p
    | $p.affected_by[]
    | { tool: "vulnix",
        id: .,
        aliases: [],
        package: $p.pname,
        version: $p.version,
        fixed: null,
        components: [],
        paths: [],
        title: ($p.description[.] // "" | first_sentence),
        score: ($p.cvssv3_basescore[.] | parse_score),
        link: ("https://nvd.nist.gov/vuln/detail/" + .) } ];

if $tool == "osv" then osv_findings
elif $tool == "vulnix" then vulnix_findings
else error("unknown tool: \($tool)") end
