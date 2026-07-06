# Finishing pass over a fully-assembled CycloneDX SBOM.
#
# Args:
#   $id         plain-identifier product subject
#   $version    product version
#   $supplier   the SBOM creator as { name, url, contact }.
#   $ownSource  the forge host/owner our own packages are published under.
#               Components sourced from here are our own and are attributed to $supplier.
#   $deployable map of store-path bom-ref -> hex SHA-256 of the deployed component.

def is_forge($url):
  $url | test("^https?://(github\\.com|gitlab\\.com|codeberg\\.org|git\\.sr\\.ht|bitbucket\\.org|gitea\\.com)/");

def has_prop($name): (.properties // []) | any(.name == $name);

def add_prop($name; $value):
  if has_prop($name) then .
  else .properties = ((.properties // []) + [ { name: $name, value: $value } ]) end;

def source_urls:
  [ .externalReferences[]? | select(.type == "vcs") | .url ]
  + [ .externalReferences[]? | select(.type == "website") | .url ];

def supplier_name($url):
  if ($url | test("^https?://")) then
    ($url | sub("^https?://"; "") | split("/") | map(select(. != ""))) as $segs
    | if   ($segs | length) >= 2 then "\($segs[0])/\($segs[1])"
      elif ($segs | length) == 1 then $segs[0]
      else null end
  else null end;

def filename:
  (.purl // "") as $p
  | if   ($p | startswith("pkg:golang")) then "\(.name)@\(.version)"
    elif ($p | startswith("pkg:cargo"))  then "\(.name)-\(.version).crate"
    else (.["bom-ref"] // .name) end;

def effective_licence:
  ([ .licenses[]? | (.expression // .license.id // .license.name) ]
   | map(select(. != null and . != ""))) as $ids
  | if ($ids | length) == 0 then null else ($ids | join(" OR ")) end;

def promote_forge_vcs:
  if (.externalReferences // []) | any(.type == "vcs") then .
  else
    ([ .externalReferences[]? | select(.type == "website" and is_forge(.url)) | .url ] | first) as $forge
    | if $forge == null then .
      else .externalReferences = ((.externalReferences // []) + [ { type: "vcs", url: $forge } ]) end
  end;

def fill_supplier:
  if (.supplier.name // null) != null then .
  else
    source_urls as $urls
    | ([ $urls[] | select(supplier_name(.) != null) ] | first) as $named
    | ($named // ($urls | first)) as $url
    | if $url == null then .
      else (supplier_name($url)) as $name
        | if $name == $ownSource then .supplier = $supplier
          else .supplier = ((.supplier // {}) + { url: [ $url ] })
            | (if $name != null then .supplier += { name: $name } else . end)
          end
      end
  end;

def bsi_taxonomy:
  add_prop("bsi:component:filename"; filename)
  | add_prop("bsi:component:executable"; "non-executable")
  | add_prop("bsi:component:archive"; "archive")
  | add_prop("bsi:component:structured"; "structured")
  | (effective_licence) as $eff
  | if $eff == null then . else add_prop("bsi:component:effectiveLicence"; $eff) end;

def ack($v):
  if has("expression") then . + { acknowledgement: $v }
  elif has("license") then .license += { acknowledgement: $v }
  else . end;

(.metadata.component["bom-ref"]) as $old
| .metadata.component = {
    type: "application",
    "bom-ref": $id,
    name: $id,
    version: $version,
    purl: ("pkg:nix/" + $id + "@" + $version),
  }
| .dependencies = ((.dependencies // []) | map(if .ref == $old then .ref = $id else . end))
| .compositions = ((.compositions // []) + [ { aggregate: "complete", dependencies: [ $id ] } ])
| .metadata.supplier = $supplier
| .metadata.manufacturer = $supplier
| .metadata.authors = ((.metadata.authors // []) + $supplier.contact)
| .components = ((.components // []) | map(
    promote_forge_vcs
    | fill_supplier
    | bsi_taxonomy
    | (($deployable[.["bom-ref"] // ""]) as $h
       | if ((.purl // "") | startswith("pkg:nix")) and ((.hashes // []) | length == 0) and $h
         then .hashes = [ { alg: "SHA-256", content: $h } ] else . end)
    | (if .licenses then .licenses |= map(ack("concluded")) else . end)
    | (if .evidence.licenses then .evidence.licenses |= map(ack("declared")) else . end)
  ))
