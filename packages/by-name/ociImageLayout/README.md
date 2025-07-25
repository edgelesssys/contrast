# OCI image tools

This is a set of nix functions for creating reproducible and cachable
multi-layer OCI images.

It uses the following functions:

- `ociImageLayout`: Top level function for creating an
  [OCI image directory layout](https://github.com/opencontainers/image-spec/blob/v1.1.0/image-layout.md).
  This can be used directly (by podman) or uploaded to a registry
  (`crane push path/to/directory registry/image/name:tag`).
- `ociImageManifest`: An
  [OCI image manifest](https://github.com/opencontainers/image-spec/blob/v1.1.0/manifest.md#image-manifest)
  that can be added to the top-level layout. A manifest contains a configuration
  and layers.
- `ociImageConfig`: An
  [OCI image configuration](https://github.com/opencontainers/image-spec/blob/v1.1.0/config.md)
  that can be included in a manifest. The configuration describes the image
  including layers, entrypoint, arguments, architecture, os and more.
- `ociLayerTar`: An
  [OCI image layer filesystem changeset (layer tar)](https://github.com/opencontainers/image-spec/blob/v1.1.0/layer.md).
  Contains an individual container image layer. Can be freely remixed with other
  layers. Takes a list of store paths and their target destinations in the
  image.

## Example

The following example creates an image containing two layers:

- one layer for nginx
- one layer with `bash`, `jq`, a script and configuration

```nix
ociImageLayout {
  manifests = [
    ociImageManifest
    {
      layers = [
        ociLayerTar {
            files = [ { source = nginx; } ];
        }
        ociLayerTar {
            files = [
             { source = bash; destination = "/bin/bash"; }
             { source = jq; destination = "/bin/jq"; }
             { source = writeShellScript "entrypoint.sh" '' jq $CONFIG_PATH ; nginx -g 'daemon off;' ''; destination = "/entrypoint.sh"; }
             { source = writers.writeJSON "conf.json" { a = 1; b = 2; }; destination = "/etc/configuration.json"; }
            ];
        }
      ];
      extraConfig = {
        "config" = {
          "Env" = [
            "PATH=/bin:/usr/bin"
            "CONFIG_PATH=/config"
          ];
          "Entrypoint" = [ "/entrypoint.sh" ];
        };
      };
      extraManifest = {
        "annotations" = {
          "org.opencontainers.image.title" = "example-image";
          "org.opencontainers.image.description" = "Example image for ociImageLayout";
        };
      };
    }
  ];
}
```
