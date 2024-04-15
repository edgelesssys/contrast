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
      tag = "v${contrast.version}";
      copyToRoot = with dockerTools; [ caCertificates ];
      config = {
        Cmd = [ "${contrast.coordinator}/bin/coordinator" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    initializer = dockerTools.buildImage {
      name = "initializer";
      tag = "v${contrast.version}";
      copyToRoot = with dockerTools; [ caCertificates ];
      config = {
        Cmd = [ "${contrast.initializer}/bin/initializer" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    openssl = dockerTools.buildImage {
      name = "openssl";
      tag = "v${contrast.version}";
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
      tag = "v${contrast.version}";
      copyToRoot = [ bash socat ];
    };

    service-mesh-proxy = dockerTools.buildImage {
      name = "service-mesh-proxy";
      tag = "v${service-mesh.version}";
      copyToRoot = [ envoy iptables-legacy ];
      config = {
        Cmd = [ "${service-mesh}/bin/service-mesh" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };
  };
in
containers // (lib.concatMapAttrs (name: container: { "push-${name}" = pushContainer container; }) containers)
