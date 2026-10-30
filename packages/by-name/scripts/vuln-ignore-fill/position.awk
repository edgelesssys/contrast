# Prints the line before which a new entry belongs in vuln-ignores.toml, or 0 if it belongs at the end.
#
# Variables:
#   table    "osv" or "vulnix"
#   package  for vulnix, the package the entry belongs to

BEGIN {
  if (table == "osv") {
    prefixes[1] = "[osv."
  } else {
    prefixes[1] = "[vulnix." package "."
    prefixes[2] = "[vulnix." package "]"
    prefixes[3] = "[vulnix.\"" package "\"."
    prefixes[4] = "[vulnix.\"" package "\"]"
  }
}

{ lines[NR] = $0 }

/^\[/ {
  matched = 0
  for (i in prefixes) {
    if (index($0, prefixes[i]) == 1) matched = 1
  }
  if (matched) {
    last = NR
    next_header = 0
  } else if (last && !next_header) {
    next_header = NR
  }
}

END {
  if (!next_header) {
    print 0
    exit
  }
  # Stay above comments that belong to the next table, and above the empty line that separates it.
  pos = next_header
  while (pos > 1 && lines[pos - 1] ~ /^#/) pos--
  if (pos > 1 && lines[pos - 1] == "") pos--
  print pos
}
