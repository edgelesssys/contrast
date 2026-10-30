# Annotates findings with the components that vendor the affected package, and how they depend on it.
#
# Inputs: the vendored per-package SBOMs, named <component>.cdx.json.
# Args:
#   $findings  slurped list of findings (see normalize.jq).

# Walks the reverse dependency graph from $targets through external packages, stopping at local ones.
# Returns { local bom-ref: shortest path of bom-refs from it to a target }.
def local_users($up; $local; $targets):
  { next: ($targets | map({ (.): null }) | add),
    frontier: ($targets | map(select($local[.] | not))),
    hits: ($targets | map(select($local[.]))) }
  | until(.frontier == [];
      reduce (.frontier[] as $n | $up[$n][]? | [ $n, . ]) as [ $child, $parent ]
        (.frontier = [];
         if .next | has($parent) then .
         else .next[$parent] = $child
           | if $local[$parent] then .hits += [ $parent ] else .frontier += [ $parent ] end
         end))
  | .next as $next
  | .hits
  | map({ (.): [ . | recurse($next[.] // empty) ] })
  | add // {};

[ inputs
  | (input_filename | split("/") | last | rtrimstr(".cdx.json")) as $component
  | .metadata.component["bom-ref"] as $root
  | { component: $component,
      root: $root,
      up: ([ .dependencies[]? | .ref as $ref | (.dependsOn // [])[] | { child: ., parent: $ref } ]
           | group_by(.child) | map({ (.[0].child): map(.parent) }) | add // {}),
      local: ([ $root, (.components[]? | select(.hashes == null) | .["bom-ref"]) ] | map({ (.): true }) | add),
      names: ([ .components[]?, .metadata.component ] | map({ (.["bom-ref"]): .name }) | add),
      refs: ([ .components[]?, .metadata.component ]
             | group_by("\(.name)@\(.version)")
             | map({ ("\(.[0].name)@\(.[0].version)"): map(.["bom-ref"]) }) | add) } ] as $boms
| $findings[0]
| map(
    "\(.package)@\(.version)" as $key
    | [ $boms[]
        | select(.refs | has($key))
        | . as $bom
        | { component,
            paths: [ local_users($bom.up; $bom.local; $bom.refs[$key])
                     | to_entries[]
                     | .value
                     | map(if . == $bom.root then $bom.component else $bom.names[.] end) ]
                   | sort_by(length, .) } ]
      as $hits
    | .components = ($hits | map(.component) | unique)
    | .paths = [ $hits[] | .component as $c | .paths[] | { component: $c, path: . } ])
