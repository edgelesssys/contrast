# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib
, stdenv
, stdenvNoCC
, distro ? "cbl-mariner"
, microsoft
, bubblewrap
, fakeroot
, fetchFromGitHub
, fetchurl
, kata-agent ? microsoft.kata-agent
, yq-go
, tdnf
, curl
, util-linux
, writeText
, writeTextDir
, createrepo_c
, writeShellApplication
, parted
, cryptsetup
, closureInfo
, erofs-utils
}:

let
  kata-version = "3.2.0.azl1";
  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "kata-containers";
    rev = kata-version;
    hash = "sha256-W36RJFf0MVRIBV4ahpv6pqdAwgRYrlqmu4Y/8qiILS8=";
  };
  # toplevelNixDeps are packages that get installed to the rootfs of the image
  # they are used to determine the (nix) closure of the rootfs
  toplevelNixDeps = [ kata-agent ];
  nixClosure = builtins.toString (lib.strings.splitString "\n" (builtins.readFile "${closureInfo {rootPaths = toplevelNixDeps;}}/store-paths"));
  rootfsExtraTree = stdenvNoCC.mkDerivation {
    inherit src;
    pname = "rootfs-extra-tree";
    version = kata-version;

    # https://github.com/microsoft/azurelinux/blob/59ce246f224f282b3e199d9a2dacaa8011b75a06/SPECS/kata-containers-cc/mariner-coco-build-uvm.sh#L34-L41
    buildPhase = ''
      runHook preBuild

      mkdir -p /build/rootfs/etc/kata-opa /build/rootfs/usr/lib/systemd/system /build/rootfs/nix/store
      cp src/agent/kata-agent.service.in /build/rootfs/usr/lib/systemd/system/kata-agent.service
      cp src/agent/kata-containers.target /build/rootfs/usr/lib/systemd/system/kata-containers.target
      cp src/kata-opa/allow-set-policy.rego /build/rootfs/etc/kata-opa/default-policy.rego
      sed -i 's/@BINDIR@\/@AGENT_NAME@/\/usr\/bin\/kata-agent/g'  /build/rootfs/usr/lib/systemd/system/kata-agent.service
      touch /build/rootfs/etc/machine-id

      tar --sort=name --mtime="@$SOURCE_DATE_EPOCH" -cvf /build/rootfs-extra-tree.tar -C /build/rootfs .

      mv /build/rootfs-extra-tree.tar $out

      runHook postBuild
    '';

    dontInstall = true;
  };
  packageIndex = builtins.fromJSON (builtins.readFile ./package-index.json);
  rpmSources = lib.forEach packageIndex
    (p: lib.concatStringsSep "#" [ (fetchurl p) (builtins.baseNameOf p.url) ]);

  mirror = stdenvNoCC.mkDerivation {
    name = "mirror";
    dontUnpack = true;
    nativeBuildInputs = [ createrepo_c ];
    buildPhase = ''
      runHook preBuild

      mkdir -p $out/packages
      for source in ${builtins.concatStringsSep " " rpmSources}; do
        path=$(echo $source | cut -d'#' -f1)
        filename=$(echo $source | cut -d'#' -f2)
        ln -s "$path" "$out/packages/$filename"
      done

      createrepo_c --revision 0 --set-timestamp-to-revision --basedir packages $out

      runHook postBuild
    '';
  };

  tdnfConf = writeText "tdnf.conf" ''
    [main]
    gpgcheck=1
    installonly_limit=3
    clean_requirements_on_remove=0
    repodir=/etc/yum.repos.d
    cachedir=/build/var/cache/tdnf
  '';
  vendor-reposdir = writeTextDir "yum.repos.d/cbl-mariner-2-vendor.repo" ''
    [cbl-mariner-2.0-prod-base-x86_64-yum]
    name=cbl-mariner-2.0-prod-base-x86_64-yum
    baseurl=file://${mirror}
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
in

stdenv.mkDerivation rec {
  inherit src;
  pname = "kata-image";
  version = kata-version;

  outputs = [ "out" "verity" ];

  env = {
    AGENT_SOURCE_BIN = "${lib.getExe kata-agent}";
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
    buildimage
  ];

  sourceRoot = "${src.name}/tools/osbuilder/rootfs-builder";

  buildPhase = ''
    runHook preBuild

    # use a fakeroot environment to build the rootfs as a tar
    # this is required to create files with the correct ownership and permissions
    # including suid
    # Upstream build invokation:
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
          --exclude='*systemd-bless-boot-generator*' \
          --exclude='*systemd-fstab-generator*' \
          --exclude='*systemd-getty-generator*' \
          --exclude='*systemd-gpt-auto-generator*' \
          --exclude='*systemd-tmpfiles-cleanup.timer*' \
          --sort=name --mtime='UTC 1970-01-01' -C /build/root -c . -f /build/rootfs.tar"

    # add the extra tree to the rootfs
    tar --concatenate --file=/build/rootfs.tar ${rootfsExtraTree}
    # add the closure to the rootfs
    tar --hard-dereference --transform 's+^+./+' -cf /build/closure.tar --mtime="@$SOURCE_DATE_EPOCH" --sort=name ${nixClosure}
    # combine the rootfs and the closure
    tar --concatenate --file=/build/rootfs.tar /build/closure.tar

    # convert tar to a squashfs image with dm-verity hash
    ${lib.getExe buildimage} /build/rootfs.tar $out

    runHook postBuild
  '';

  postInstall = ''
    # split outputs into raw image (out) and dm-verity data (verity)
    mkdir -p $verity
    mv $out/{dm_verity.txt,roothash,salt,hash_type,data_blocks,data_block_size,hash_blocks,hash_block_size,hash_algorithm} $verity/
    mv $out/raw.img /build/raw.img
    rm -rf $out
    mv /build/raw.img $out
  '';
}
