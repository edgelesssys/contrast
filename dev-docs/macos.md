# Development setup for macOS (experimental)

Contrast uses `just` and Nix as its build system. Several packages Contrast needs to build, such as container images (`nix build .#base.containers.*`), need to be built for `x86_64-linux` so when building from a different architecture such as `aarch64-darwin`, those builds need to be delegated to a builder that can build for `x86_64-linux`.

## Canonical setup

1. Install Nix. You have several options such the [Lix installer](https://lix.systems/install/) (recommended), the [Determinate Nix installer](https://docs.determinate.systems/) or by following the [official instructions](https://nixos.org/download/). It's recommended to use one of the automated installers as they also make the uninstall on macOS easy.

2. Setup a `x86_64-linux` builder. There are 2 options:

   - setup a remote builder by following Nix's [distributed builds tutorial](https://nix.dev/tutorials/nixos/distributed-builds-setup.html). If you are working for Edgeless Systems, you can use one of our office machines by following the instructions in https://github.com/edgelesssys/nix-remote-builders.
   - setup a local VM-based builder that emulates x86, by installing [nix-rosetta-builder](https://github.com/cpick/nix-rosetta-builder). Not that the performance of this option might not be great but it's helpful if you need to work offline.

It's recommended to setup both. Nix will automatically offload packages that need to be built for `x86_64-linux` to any builder available for that architecture. So if one of the remote machines isn't available, builds will use the VM-based builder.

## Alternative setup using a Linux VM

Alternatively you can setup a VM with Nix which you can use to build contrast. Since this option will be also using emulation, the performance might not be great.

1. Follow the instructions on [nixos-lima](https://github.com/nixos-lima/nixos-lima) and [nixos-lima-config-sample](https://github.com/nixos-lima/nixos-lima-config-sample) to create a `x86_64-linux` VM.

2. To avoid having to authenticate twice either with your container registry or kubectl, you can forward the local credentials to the VM by adding the following in the VM configuration:

   ```yaml
   - location: "~/.docker"
     mountPoint: "/home/lima.linux/.docker"
     writable: true
     9p:
       cache: "mmap"
   - location: "~/.kube"
     mountPoint: "/home/lima.linux/.kube"
     writable: true
   ```

3. Forward contrast project path as well:

   ```yaml
   - location: "~/contrast"
     writable: true
     9p:
       cache: "mmap"
   ```

4. Add the lima user to trusted-users by adding the following in the VM's NixOS configuration (`configuration.nix`):

   ```nix
   nix.settings.trusted-users = [ "root" "@wheel" ];
   ```

5. (Optional) You might have to add the hosts you are deploying to in the VM's NixOS configuration:

   ```nix
   networking.hosts = {
     "XXX.YYY.ZZZ.XXX" = [ "<SOME HOSTNAME>" ];
   };
   ```

6. Start a `x86_64` VM with:

   ```bash
   limactl start --yes --set '.user.name = "lima"' nixos.yaml --arch=x86_64
   ```

7. Connect to the VM with:

   ```
   cd ~/contrast
   limactl shell nixos
   nix develop .#
   ```
