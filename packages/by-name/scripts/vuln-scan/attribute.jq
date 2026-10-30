# Annotates findings with the components that vendor the affected package.
#
# Inputs: the vendored per-package SBOMs, named <component>.cdx.json.
# Args:
#   $findings  slurped list of findings (see normalize.jq).

(reduce inputs as $bom ({};
  (input_filename | split("/") | last | rtrimstr(".cdx.json")) as $component
  | reduce ($bom.components[]?, $bom.metadata.component | "\(.name)@\(.version)") as $key
      (.; .[$key] += [ $component ]))) as $index
| $findings[0]
| map(.components = ($index["\(.package)@\(.version)"] // [] | unique))
