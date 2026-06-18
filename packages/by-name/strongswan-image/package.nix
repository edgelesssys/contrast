# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  runCommand,
}:

runCommand "strongswan-rootfs" { } ''
  mkdir -p \
    $out/etc/strongswan.d \
    $out/etc/swanctl/x509 \
    $out/etc/swanctl/private \
    $out/etc/swanctl/x509ca \
    $out/var/run

  cp -r ${./strongswan.conf} $out/etc/strongswan.conf

  ln -s /contrast/tls-config/certChain.pem $out/etc/swanctl/x509/certChain.pem
  ln -s /contrast/tls-config/key.pem $out/etc/swanctl/private/key.pem
  ln -s /contrast/tls-config/mesh-ca.pem $out/etc/swanctl/x509ca/mesh-ca.pem
''
