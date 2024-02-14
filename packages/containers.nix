{ pkgs }:

with pkgs;

let
  pushContainer = container: writeShellApplication {
    name = "push-${container.name}";
    runtimeInputs = [ crane gzip ];
    text = ''
      imageName="$1"
      tmpdir=$(mktemp -d)
      trap 'rm -rf $tmpdir' EXIT
      gunzip < "${container}" > "$tmpdir/image.tar"
      crane push "$tmpdir/image.tar" "$imageName:${container.imageTag}"
    '';
  };

  containers = {
    coordinator = dockerTools.buildImage {
      name = "coordinator";
      tag = "v${nunki.version}";
      copyToRoot = with dockerTools; [ caCertificates ];
      config = {
        Cmd = [ "${nunki.coordinator}/bin/coordinator" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    initializer = dockerTools.buildImage {
      name = "initializer";
      tag = "v${nunki.version}";
      copyToRoot = with dockerTools; [ caCertificates ];
      config = {
        Cmd = [ "${nunki.initializer}/bin/initializer" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    openssl = dockerTools.buildImage {
      name = "openssl";
      tag = "v${nunki.version}";
      copyToRoot = [
        bash
        bashInteractive
        coreutils
        ncurses
        openssl
        procps
        vim
      ];
      config = {
        Cmd = [ "bash" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    port-forwarder = dockerTools.buildImage {
      name = "port-forwarder";
      tag = "v${nunki.version}";
      copyToRoot = [ bash socat ];
    };
  };
in
containers // (lib.concatMapAttrs (name: container: { "push-${name}" = pushContainer container; }) containers)
