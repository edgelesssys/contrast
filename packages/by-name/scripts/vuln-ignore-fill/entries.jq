# Renders vuln-ignores.toml entries for findings (see vuln-scan's normalize.jq), with an empty reason.
#
# Input: the slurped findings files.
# Outputs one { table, package, block } per entry. Each block starts with an empty line.

def short: if length > 120 then .[:119] + "..." else . end;

# Package names and ids are used as table keys, which need quotes unless they are bare keys.
def key: if test("^[A-Za-z0-9_-]+$") then . else tojson end;

# JSON strings are valid TOML basic strings.
def field($key; $value): if $value == null or $value == "" then empty else "\($key) = \($value | tojson)" end;

add
# osv matches by id, so an advisory affecting several packages or versions needs only one entry.
| (map(select(.tool == "osv"))
   | group_by(.id)[]
   | { table: "osv",
       package: null,
       block: ([ "",
                 "[osv.\(.[0].id | key)]",
                 field("package"; map(.package) | unique | join(", ")),
                 field("title"; .[0].title | short),
                 field("link"; .[0].link),
                 "reason = \"\"" ] | join("\n")) }),
  # vulnix matches by package and CVE id.
  (map(select(.tool == "vulnix"))
   | group_by([ .package, .id ])[]
   | { table: "vulnix",
       package: .[0].package,
       block: ([ "",
                 "[vulnix.\(.[0].package | key).\(.[0].id | key)]",
                 field("title"; .[0].title | short),
                 field("link"; .[0].link),
                 "reason = \"\"" ] | join("\n")) })
