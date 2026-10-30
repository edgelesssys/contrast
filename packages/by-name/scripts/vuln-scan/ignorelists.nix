# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  formats,
}:

rec {
  entries = fromTOML (builtins.readFile ../../../../vuln-ignores.toml);

  checked =
    name: allowedKeys: e:
    lib.throwIf (lib.subtractLists allowedKeys (builtins.attrNames e) != [ ])
      "vuln-ignores.toml: ${name} has unknown keys ${toString (lib.subtractLists allowedKeys (builtins.attrNames e))}"
      (lib.throwIf ((e.reason or "") == "") "vuln-ignores.toml: ${name} needs a reason" e);

  osvConfig = (formats.toml { }).generate "osv-scanner.toml" {
    IgnoredVulns = lib.mapAttrsToList (id: e: {
      inherit id;
      inherit
        (checked "[osv.${id}]" [
          "package"
          "title"
          "link"
          "reason"
          "drop_when"
        ] e)
        reason
        ;
    }) (entries.osv or { });
  };

  cvesOf = lib.filterAttrs (_: builtins.isAttrs);

  vulnixSection =
    package: p:
    if cvesOf p == { } then
      {
        comment = [
          (checked "[vulnix.${package}]" [
            "note"
            "reason"
            "drop_when"
          ] p).reason
        ];
      }
    else
      lib.throwIf (p ? reason)
        "vuln-ignores.toml: [vulnix.${package}] lists CVEs, so it can't have a reason that ignores all of them"
        (
          lib.throwIf
            (
              lib.subtractLists [ "note" ] (builtins.attrNames (lib.filterAttrs (_: v: !builtins.isAttrs v) p))
              != [ ]
            )
            "vuln-ignores.toml: [vulnix.${package}] can only have a note besides its CVEs"
            {
              cve = builtins.attrNames (cvesOf p);
              comment = lib.mapAttrsToList (
                id: e:
                "${id}: ${
                  (checked "[vulnix.${package}.${id}]" [
                    "title"
                    "link"
                    "reason"
                    "drop_when"
                  ] e).reason
                }"
              ) (cvesOf p);
            }
        );

  vulnixWhitelist = (formats.toml { }).generate "vulnix-whitelist.toml" (
    lib.mapAttrs vulnixSection (entries.vulnix or { })
  );
}
