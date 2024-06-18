## Contributing

Thank you for getting involved! Before you start, please familiarize yourself with the [documentation](https://docs.edgeless.systems/contrast).

Please follow our [Code of Conduct](CODE_OF_CONDUCT.md) when interacting with this project.

### Opening an issue, starting a discussion, asking a question

This project uses the GitHub issue tracker. Check the existing issues before submitting to avoid duplicates.

If you have a broader topic to discuss or a question, please [open a discussion](https://github.com/edgelesssys/contrast/discussions) instead.

### Pull requests

Contrast is licensed under the [AGPLv3](LICENSE). When contributing, you also need to agree to our
[Contributor License Agreement](https://cla-assistant.io/edgelesssys/contrast).

### Development setup

1. [Install Nix](https://zero-to-nix.com/concepts/nix-installer)

2. (Optional) configure Nix to allow use of extra substituters, and profit from our
    cachix remote cache. To allow using additional substituters from the `flake.nix`,
    add your user name (or the wheel group) as trusted-user in your nix config.

    On NixOS (in your config):

    ```nix
    nix.settings.trusted-users = [ "root" "@wheel" ];
    ```

    On other systems (in `/etc/nix/nix.conf`):

    ```
    trusted-users = root @wheel
    ```

    See Nix manual section on [substituters](https://nixos.org/manual/nix/stable/command-ref/conf-file.html#conf-substituters)
    and [trusted-users](https://nixos.org/manual/nix/stable/command-ref/conf-file.html#conf-trusted-users) for details and
    consequences.

3. Enter the development environment with

    ```sh
    nix develop .#
    ```

   Or activate [`direnv`](https://direnv.net/) to automatically enter the nix shell.
   It's recommended to use [`nix-direnv`](https://github.com/nix-community/nix-direnv).
   If your system ships outdated bash, [install `direnv`](https://direnv.net/docs/installation.html) via package manager.
   Additionally, you may want to add the [VSCode extension](https://github.com/direnv/direnv-vscode).

   ```sh
   direnv allow
   ```

4. Execute and follow instructions of

    ```sh
    just onboard
    ```

5. Provision a CoCo enabled AKS cluster with

    ```sh
    just create
    ```

    The kubeconfig of the cluster will be automatically downloaded and merged with your default config.
    You can get the kubeconfig of the running cluster at a later time with

    ```sh
    just get-credentials
    ```

### Deploy

The usual developer flow is available as a single target to execute:

```sh
just [default <deployment-name>]
```

This will build, containerize and push all relevant components.
Ensure the pushed container images are accessible to your cluster.
The manifest will the be generated (`contrast generate`).

Further the flow will deploy the selected deployment and wait for components to come up.
The manifest will automatically be set (`contrast set`) and the Coordinator will be verified
(`contrast verify`). The flow will also wait for the workload to get ready.

This target is idempotent and will delete an existing deployment before re-deploying.

All steps can be executed as separate targets. To list all available targets and their description, run

```sh
just --list
```

### Cleanup

- Destroy the cluster with

    ```sh
    just destroy
    ```

### Maintenance tasks

- Run code generation

    ```sh
    just codegen
    ```

- Format all code

    ```sh
    nix fmt
    ```
