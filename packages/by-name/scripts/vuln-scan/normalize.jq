# Normalizes the JSON output of osv-scanner or vulnix into a common list of findings.
#
# Args:
#   $tool  "osv" or "vulnix", selecting the input format.
#
# Each finding is { tool, id, aliases, package, version, fixed, components, title, score, link }.
# components is filled in later by attribute.jq.

# Shortens NVD descriptions, which are paragraphs rather than titles, to their first sentence.
def first_sentence:
  (capture("^(?<s>.*?\\.)(\\s|$)").s // .);

def score: if . == null or . == "" then null else tonumber end;

# Comparable form of a version: "v1.2.3-rc.1" -> [1, 2, 3].
def vparts: ltrimstr("v") | sub("[-+].*$"; "") | split(".") | map(tonumber? // 0);

# Advisories sometimes spell crate names with "_" where the SBOM has "-".
def canon: ascii_downcase | gsub("_"; "-");

def osv_findings:
  [ .results[]?.packages[]? as $p
    | ($p.vulnerabilities | map({ (.id): . }) | add // {}) as $vulns
    | ($p.package.version | vparts) as $current
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
                           and (.package.name | canon) == ($p.package.name | canon))
                  | .ranges[]?.events[]?.fixed // empty ]
                | unique
                | map(select(vparts > $current))
                | min_by(vparts)),
        components: [],
        title: ($vulns[$id].summary // ""),
        score: (.max_severity | score),
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
        title: ($p.description[.] // "" | first_sentence),
        score: ($p.cvssv3_basescore[.] | score),
        link: ("https://nvd.nist.gov/vuln/detail/" + .) } ];

if $tool == "osv" then osv_findings
elif $tool == "vulnix" then vulnix_findings
else error("unknown tool: \($tool)") end
