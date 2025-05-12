# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  stdenv,
  stdenvNoCC,
  distro ? "cbl-mariner",
  microsoft,
  bubblewrap,
  fakeroot,
  fetchurl,
  yq-go,
  tdnf,
  curl,
  util-linux,
  writeText,
  writeTextDir,
  createrepo_c,
  writeShellApplication,
  parted,
  cryptsetup,
  closureInfo,
  erofs-utils,
  gnutar,
  runCommand,
}:

let
  nixClosure =
    let
      # toplevelNixDeps are packages that get installed to the rootfs of the image
      # they are used to determine the (nix) closure of the rootfs
      toplevelNixDeps = [ microsoft.kata-agent ];
    in
    builtins.toString (
      lib.strings.splitString "\n" (
        builtins.readFile "${closureInfo { rootPaths = toplevelNixDeps; }}/store-paths"
      )
    );
  closureTar = runCommand "closure.tar" { nativeBuildInputs = [ gnutar ]; } ''
    tar --hard-dereference --transform 's+^+./+' -cf $out --mtime="@$SOURCE_DATE_EPOCH" --sort=name ${nixClosure}
  '';

  rootfsExtraTree = stdenvNoCC.mkDerivation {
    pname = "rootfs-extra-tree";
    inherit (microsoft.genpolicy) src version;

    # https://github.com/microsoft/azurelinux/blob/59ce246f224f282b3e199d9a2dacaa8011b75a06/SPECS/kata-containers-cc/mariner-coco-build-uvm.sh#L34-L41
    buildPhase = ''
      runHook preBuild

      mkdir -p /build/rootfs/etc/kata-opa /build/rootfs/usr/lib/systemd/system /build/rootfs/nix/store
      cp src/agent/kata-agent.service.in /build/rootfs/usr/lib/systemd/system/kata-agent.service
      cp src/agent/kata-containers.target /build/rootfs/usr/lib/systemd/system/kata-containers.target
      cp src/kata-opa/allow-set-policy.rego /build/rootfs/etc/kata-opa/default-policy.rego
      sed -i 's/@BINDIR@\/@AGENT_NAME@/\/usr\/bin\/kata-agent/g'  /build/rootfs/usr/lib/systemd/system/kata-agent.service
      touch /build/rootfs/etc/machine-id

      # For more information about this unit see packages/by-name/mkNixosConfig/package.nix
      cat > /build/rootfs/usr/lib/systemd/system/deny-incoming-traffic.service<< EOF
      [Unit]
      Description="Deny all incoming connections"

      Wants=network.target
      After=network.target
      Before=kata-agent.service

      [Service]
      Type=oneshot
      RemainAfterExit=yes
      ExecStart=iptables-legacy -I INPUT -m conntrack ! --ctstate ESTABLISHED,RELATED -j DROP
      EOF

      # create a "RequiredBy" dependency where the kata-agent.service requires
      # the deny-incoming-traffic.service to be started and successfully exited
      # before the kata-agent.service can be started
      # Usually this is done via the "RequiredBy" directive under the [Install]
      # section of the systemd unit file, but the filesystem is read only
      # and systemd wants the create the following symlink.
      mkdir -p /build/rootfs/etc/systemd/system/kata-agent.service.requires/
      ln -s ../../../../../usr/lib/systemd/system/deny-incoming-traffic.service /build/rootfs/etc/systemd/system/kata-agent.service.requires/deny-incoming-traffic.service


      tar --sort=name --mtime="@$SOURCE_DATE_EPOCH" -cvf /build/rootfs-extra-tree.tar -C /build/rootfs .

      mv /build/rootfs-extra-tree.tar $out

      runHook postBuild
    '';
    dontInstall = true;
    dontPatchELF = true;
  };

  packageIndex = builtins.fromJSON (builtins.readFile ./package-index.json);
  rpmSources = lib.forEach packageIndex (
    p:
    lib.concatStringsSep "#" [
      (fetchurl p)
      (builtins.baseNameOf p.url)
    ]
  );
  rpmMirror = runCommand "rpm-mirror" { nativeBuildInputs = [ createrepo_c ]; } ''
    mkdir -p $out/packages
    for source in ${builtins.concatStringsSep " " rpmSources}; do
      path=$(echo $source | cut -d'#' -f1)
      filename=$(echo $source | cut -d'#' -f2)
      ln -s "$path" "$out/packages/$filename"
    done

    createrepo_c --revision 0 --set-timestamp-to-revision --basedir packages $out
  '';
  tdnfConf = writeText "tdnf.conf" ''
    [main]
    gpgcheck=1
    installonly_limit=3
    clean_requirements_on_remove=0
    repodir=/etc/yum.repos.d
    cachedir=/build/var/cache/tdnf
  '';
  vendor-reposdir = writeTextDir "yum.repos.d/azurelinux-3.0-vendor.repo" ''
    [azurelinux-3.0-prod-base-x86_64]
    name=azurelinux-3.0-prod-base-x86_64
    baseurl=file://${rpmMirror}
    repo_gpgcheck=0
    gpgcheck=0
    enabled=1
  '';

  buildimage = writeShellApplication {
    name = "buildimage";
    runtimeInputs = [
      parted
      erofs-utils
      cryptsetup
    ];
    text = builtins.readFile ./buildimage.sh;
  };

  rootfsTar = stdenv.mkDerivation rec {
    pname = "kata-image-rootfs.tar";
    inherit (microsoft.genpolicy) src version;

    env = {
      AGENT_SOURCE_BIN = "${lib.getExe microsoft.kata-agent}";
      CONF_GUEST = "yes";
      RUST_VERSION = "not-needed";
    };

    nativeBuildInputs = [
      yq-go
      curl
      fakeroot
      bubblewrap
      util-linux
      tdnf
    ];

    sourceRoot = "${src.name}/tools/osbuilder/rootfs-builder";

    buildPhase = ''
      runHook preBuild

      # use a fakeroot environment to build the rootfs as a tar
      # this is required to create files with the correct ownership and permissions
      # including suid
      # Upstream build invocation:
      # https://github.com/microsoft/azurelinux/blob/59ce246f224f282b3e199d9a2dacaa8011b75a06/SPECS/kata-containers-cc/mariner-coco-build-uvm.sh#L18
      mkdir -p /build/var/run
      mkdir -p /build/var/tdnf
      mkdir -p /build/var/lib/tdnf
      mkdir -p /build/var/cache/tdnf
      mkdir -p /build/root
      unshare --map-root-user bwrap \
        --bind /nix /nix \
        --bind ${tdnfConf} /etc/tdnf/tdnf.conf \
        --bind ${vendor-reposdir}/yum.repos.d /etc/yum.repos.d \
        --bind /build /build \
        --bind /build/var /var \
        --dev-bind /dev/null /dev/null \
        fakeroot bash -c "bash $(pwd)/rootfs.sh -r /build/root ${distro} && \
          tar \
            --exclude='./usr/lib/systemd/system/systemd-coredump@*' \
            --exclude='./usr/lib/systemd/system/systemd-journald*' \
            --exclude='./usr/lib/systemd/system/systemd-journald-dev-log*' \
            --exclude='./usr/lib/systemd/system/systemd-journal-flush*' \
            --exclude='./usr/lib/systemd/system/systemd-random-seed*' \
            --exclude='./usr/lib/systemd/system/systemd-timesyncd*' \
            --exclude='./usr/lib/systemd/system/systemd-tmpfiles-setup*' \
            --exclude='./usr/lib/systemd/system/systemd-update-utmp*' \
            --exclude='./etc/udev/hwdb.bin' \
            --exclude='./etc/machine-id' \
            --exclude='./nix/store' \
            --exclude='*systemd-bless-boot-generator*' \
            --exclude='*systemd-fstab-generator*' \
            --exclude='*systemd-getty-generator*' \
            --exclude='*systemd-gpt-auto-generator*' \
            --exclude='*systemd-tmpfiles-cleanup.timer*' \
            --sort=name --mtime='UTC 1970-01-01' -C /build/root -c . -f $out"

      runHook postBuild
    '';

    dontPatchELF = true;
  };

  rootfsCombinedTar = runCommand "rootfs-combined.tar" { nativeBuildInputs = [ gnutar ]; } ''
    cp ${rootfsTar} $out
    chmod +w $out
    tar --concatenate --file=$out ${rootfsExtraTree}
    tar --concatenate --file=$out ${closureTar}
  '';
in

stdenv.mkDerivation {
  pname = "kata-image";
  inherit (microsoft.genpolicy) version;

  dontUnpack = true;

  outputs = [
    "out"
    "verity"
  ];

  nativeBuildInputs = [
    buildimage
    util-linux
  ];

  buildPhase = ''
    runHook preBuild

    ${lib.getExe buildimage} ${rootfsCombinedTar} .

    runHook postBuild
  '';

  postInstall = ''
    # split outputs into raw image (out) and dm-verity data (verity)
    mkdir -p $verity
    mv dm_verity.txt \
      roothash \
      salt \
      hash_type \
      data_blocks \
      data_block_size \
      hash_blocks \
      hash_block_size \
      hash_algorithm \
      $verity/
    mv raw.img $out
  '';
  dontPatchELF = true;

  passthru = {
    inherit
      rootfsTar
      closureTar
      rootfsExtraTree
      rootfsCombinedTar
      ;
  };
}
